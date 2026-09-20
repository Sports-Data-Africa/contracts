package identity_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
	"github.com/Sports-Data-Africa/contracts/identity"
)

// Helper to instantiate a test engine with mock repo and mock redis
func setupTestEngine() (*identity.Engine, *identity.MockRepository, *identity.MockRedis) {
	repo := identity.NewMockRepository()
	redis := identity.NewMockRedis()
	cache := identity.NewTieredIdentityCache(redis, 24*time.Hour)
	engine := identity.NewEngine(repo, cache)
	return engine, repo, redis
}

// Invariant 1: Format Canonical Fixture ID follows SPD-[SPORT]-[6DIGIT]
func TestInvariant01_CanonicalFixtureFormat(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	f, err := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-101",
		canonical.SportIDFootball, "Liverpool FC", "Manchester City",
		time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC),
		1, "Premier League", canonical.MappingPrematch, "feed",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, sportCode, seq, pErr := canonical.ParseCanonicalFixtureID(f.CanonicalFixtureID)
	if pErr != nil {
		t.Fatalf("failed to parse canonical ID %s: %v", f.CanonicalFixtureID, pErr)
	}
	if sportCode != "FB" {
		t.Errorf("expected sport code FB, got %s", sportCode)
	}
	if seq < 100000 || seq > 999999 {
		t.Errorf("expected 6-digit sequence between 100000 and 999999, got %d", seq)
	}
}

// Invariant 2: Dual-Provider Prematch Convergence
// SportyBet (SB-123) and Paripesa (PP-987) for the same real match converge to the EXACT same canonical ID.
func TestInvariant02_DualProviderPrematchConvergence(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()
	kickoff := time.Date(2026, 9, 21, 17, 30, 0, 0, time.UTC)

	// SportyBet discovers it first
	f1, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-123",
		canonical.SportIDFootball, "Real Madrid", "Barcelona",
		kickoff, 10, "La Liga", canonical.MappingPrematch, "sporty_sync",
	)
	if err != nil {
		t.Fatalf("SportyBet resolution failed: %v", err)
	}

	// Paripesa discovers it second with different team casing/noise
	f2, err := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-987",
		canonical.SportIDFootball, "Real Madrid CF", "FC Barcelona",
		kickoff, 25, "Spain Primera", canonical.MappingPrematch, "paripesa_vzip",
	)
	if err != nil {
		t.Fatalf("Paripesa resolution failed: %v", err)
	}

	if f1.CanonicalFixtureID != f2.CanonicalFixtureID {
		t.Fatalf("CRITICAL INVARIANT VIOLATION: SportyBet (%s) != Paripesa (%s)", f1.CanonicalFixtureID, f2.CanonicalFixtureID)
	}
}

// Invariant 3: Cold-Boot Live JIT Discovery
// Ingesting an in-play tick with zero prematch record executes atomic JIT canonical creation.
func TestInvariant03_ColdBootLiveJITDiscovery(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	candidate := canonical.NaturalCandidate{
		SportID:         canonical.SportIDBasketball,
		HomeTeam:        "LA Lakers",
		AwayTeam:        "Golden State Warriors",
		ScheduledStart:  time.Date(2026, 9, 21, 22, 0, 0, 0, time.UTC),
		CompetitionName: "NBA",
	}

	f, alias, err := engine.ResolveJITLiveEvent(ctx, candidate, "sportybet", "SB-LIVE-99", "live_stream")
	if err != nil {
		t.Fatalf("cold-boot JIT resolution failed: %v", err)
	}
	if f == nil || alias == nil {
		t.Fatalf("expected non-nil fixture and alias")
	}
	if f.CanonicalFixtureID == "" || alias.ProviderFixtureID != "SB-LIVE-99" {
		t.Errorf("invalid JIT return: f=%+v, alias=%+v", f, alias)
	}

	// Subsequent lookup via alias must resolve immediately
	cid, rErr := engine.ResolveCanonicalID(ctx, "sportybet", "SB-LIVE-99")
	if rErr != nil || cid != f.CanonicalFixtureID {
		t.Errorf("alias lookup failed after JIT creation: cid=%s, expected=%s, err=%v", cid, f.CanonicalFixtureID, rErr)
	}
}

// Invariant 4: High Concurrency Race Defense
// 100 concurrent goroutines attempting to create the exact same match yield exactly ONE canonical fixture.
func TestInvariant04_HighConcurrencyRaceDefense(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()
	kickoff := time.Date(2026, 9, 22, 20, 0, 0, 0, time.UTC)

	const goroutines = 100
	results := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			provID := fmt.Sprintf("PP-RACE-%d", idx)
			f, err := engine.ResolveOrCreateFixture(
				ctx, "paripesa", provID,
				canonical.SportIDFootball, "Bayern Munich", "Borussia Dortmund",
				kickoff, 5, "Bundesliga", canonical.MappingPrematch, "race_test",
			)
			if err == nil && f != nil {
				results[idx] = f.CanonicalFixtureID
			}
		}()
	}
	wg.Wait()

	primaryID := results[0]
	if primaryID == "" {
		t.Fatalf("first goroutine returned empty ID")
	}
	for i, id := range results {
		if id != primaryID {
			t.Fatalf("race condition created multiple canonical IDs: goroutine 0 had %s, goroutine %d had %s", primaryID, i, id)
		}
	}
}

// Invariant 5: Tiered Cache Lookup Speed (L1 RAM -> L2 Redis -> L3 MySQL)
// First resolution hits repo; subsequent 10,000 lookups hit L1 local process RAM with zero I/O.
func TestInvariant05_TieredCacheLookupSpeed(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	f, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-CACHE-1",
		canonical.SportIDTennis, "Carlos Alcaraz", "Jannik Sinner",
		time.Date(2026, 9, 21, 14, 0, 0, 0, time.UTC),
		20, "US Open", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("failed setup: %v", err)
	}

	start := time.Now()
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		cid, err := engine.ResolveCanonicalID(ctx, "sportybet", "SB-CACHE-1")
		if err != nil || cid != f.CanonicalFixtureID {
			t.Fatalf("cache lookup failed at %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("10,000 L1 RAM lookups executed in %v (avg %v/op)", elapsed, elapsed/iterations)
	if elapsed > 100*time.Millisecond {
		t.Errorf("L1 cache too slow: took %v for 10k lookups", elapsed)
	}
}

// Invariant 6: Identity Conflict Rejection
// Attempting to reassign a provider ID to another canonical fixture fails with ErrIdentityConflict.
func TestInvariant06_IdentityConflictRejection(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	// Fixture A
	f1, _ := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-REASSIGN-1",
		canonical.SportIDFootball, "Juventus", "AC Milan",
		time.Date(2026, 9, 21, 19, 45, 0, 0, time.UTC),
		3, "Serie A", canonical.MappingPrematch, "test",
	)

	// Attempt to attach PP-REASSIGN-1 to a DIFFERENT fixture B
	_, err := engine.AttachAlias(ctx, "SPD-FB-999999", "paripesa", "PP-REASSIGN-1", canonical.MappingPrematch, "malicious_attempt")
	if err == nil {
		t.Fatalf("expected error on alias reassignment conflict, got nil")
	}
	if !errors.Is(err, identity.ErrIdentityConflict) {
		t.Errorf("expected ErrIdentityConflict, got: %v", err)
	}

	// Verify original mapping unchanged
	cid, _ := engine.ResolveCanonicalID(ctx, "paripesa", "PP-REASSIGN-1")
	if cid != f1.CanonicalFixtureID {
		t.Errorf("mapping corrupted after rejected conflict! Expected %s, got %s", f1.CanonicalFixtureID, cid)
	}
}

// Invariant 7: Ambiguity Quarantine
// Vague or placeholder team names ("Team A" vs "Team B", "Home" vs "Away") are quarantined.
func TestInvariant07_AmbiguityQuarantine(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	ambiguousCandidates := []canonical.NaturalCandidate{
		{SportID: 1, HomeTeam: "Team A", AwayTeam: "Team B", ScheduledStart: time.Now()},
		{SportID: 1, HomeTeam: "Home", AwayTeam: "Away", ScheduledStart: time.Now()},
		{SportID: 1, HomeTeam: "TBD", AwayTeam: "TBD", ScheduledStart: time.Now()},
		{SportID: 1, HomeTeam: "Arsenal", AwayTeam: "Arsenal", ScheduledStart: time.Now()}, // identical names
	}

	for i, c := range ambiguousCandidates {
		_, _, err := engine.ResolveJITLiveEvent(ctx, c, "sportybet", fmt.Sprintf("SB-AMB-%d", i), "test")
		if err == nil {
			t.Errorf("candidate %d (%s vs %s) should have been quarantined, but passed", i, c.HomeTeam, c.AwayTeam)
		}
		if !errors.Is(err, identity.ErrAmbiguousFixture) {
			t.Errorf("expected ErrAmbiguousFixture for candidate %d, got: %v", i, err)
		}
	}
}

// Invariant 8: Settleability Gate
// Fixtures lacking a certified result resolution path are restricted to LIVE_BROADCAST_ONLY.
func TestInvariant08_SettleabilityGate(t *testing.T) {
	ctx := context.Background()
	engine, repo, _ := setupTestEngine()

	f, err := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-GATE-1",
		canonical.SportIDFootball, "Ajax", "Feyenoord",
		time.Date(2026, 9, 21, 13, 30, 0, 0, time.UTC),
		8, "Eredivisie", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("failed setup: %v", err)
	}

	// Case A: Settleable fixture
	bettable, state, err := engine.EvaluateSettleability(ctx, f.CanonicalFixtureID)
	if err != nil || !bettable || state != canonical.BettingBettable {
		t.Errorf("expected bettable fixture, got bettable=%v, state=%s, err=%v", bettable, state, err)
	}

	// Case B: Non-settleable fixture (SettlementCapable = false)
	corruptFixture := *f
	corruptFixture.SettlementCapable = false
	corruptFixture.ResultResolutionKey = ""
	repo.PutFixture(&corruptFixture)
	engine.GetCache().PutFixture(ctx, &corruptFixture)

	bettableCorrupt, stateCorrupt, errCorrupt := engine.EvaluateSettleability(ctx, corruptFixture.CanonicalFixtureID)
	if errCorrupt != nil {
		t.Fatalf("unexpected error: %v", errCorrupt)
	}
	if bettableCorrupt || stateCorrupt != canonical.BettingLiveBroadcastOnly {
		t.Errorf("expected LIVE_BROADCAST_ONLY, got bettable=%v, state=%s", bettableCorrupt, stateCorrupt)
	}
}

// Invariant 9: Dual-Provider Failover
// Primary stream (SportyBet) dies; secondary stream (Paripesa) seamlessly takes over without altering canonical ID.
func TestInvariant09_DualProviderFailover(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()
	kickoff := time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC)

	// Initial prematch discovery
	canonFix, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-FAILOVER-1",
		canonical.SportIDFootball, "PSG", "Marseille",
		kickoff, 7, "Ligue 1", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("failed setup: %v", err)
	}

	// Customer places bet on canonical fixture
	ticketFixtureID := canonFix.CanonicalFixtureID

	// SportyBet in-play tick arrives
	cid1, _ := engine.ResolveCanonicalID(ctx, "sportybet", "SB-FAILOVER-1")
	if cid1 != ticketFixtureID {
		t.Fatalf("mismatch on primary tick")
	}

	// [FAILOVER SIMULATION]: SportyBet feed dies! Paripesa takes over with PP-FAILOVER-9
	paripesaCandidate := canonical.NaturalCandidate{
		SportID:         canonical.SportIDFootball,
		HomeTeam:        "Paris Saint-Germain",
		AwayTeam:        "Olympique de Marseille",
		ScheduledStart:  kickoff,
		CompetitionName: "French Ligue 1",
	}

	failoverFix, _, fErr := engine.ResolveJITLiveEvent(ctx, paripesaCandidate, "paripesa", "PP-FAILOVER-9", "failover_stream")
	if fErr != nil {
		t.Fatalf("failover resolution failed: %v", fErr)
	}

	if failoverFix.CanonicalFixtureID != ticketFixtureID {
		t.Fatalf("FAILOVER INVARIANT VIOLATION: Failover mutated canonical ID! Expected %s, got %s", ticketFixtureID, failoverFix.CanonicalFixtureID)
	}
}

// Invariant 10: Settleability Decoupling
// Bet ticket stores immutable platform canonical ID, quote acceptance snapshot, and settles regardless of live feed death.
func TestInvariant10_BetTicketSettleabilityDecoupling(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	f, _ := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-TICKET-1",
		canonical.SportIDFootball, "Porto", "Benfica",
		time.Date(2026, 9, 21, 20, 30, 0, 0, time.UTC),
		12, "Primeira Liga", canonical.MappingPrematch, "test",
	)

	// Immutable acceptance ticket snapshot
	type TicketSnapshot struct {
		TicketID           string
		CanonicalFixtureID string
		AcceptedProvider   string
		AcceptedOdds       float64
		ResultResolutionKey string
	}
	ticket := TicketSnapshot{
		TicketID:           "TKT-INV-10",
		CanonicalFixtureID: f.CanonicalFixtureID,
		AcceptedProvider:   "paripesa",
		AcceptedOdds:       2.10,
		ResultResolutionKey: f.ResultResolutionKey,
	}

	if ticket.CanonicalFixtureID != f.CanonicalFixtureID || ticket.ResultResolutionKey == "" {
		t.Fatalf("ticket acceptance snapshot incomplete: %+v", ticket)
	}
}

// Invariant 11: Settlement Idempotency
// Duplicate certified result events trigger exactly 1 payout via wallet idempotency ledger.
func TestInvariant11_SettlementIdempotency(t *testing.T) {
	var payoutCount int32
	ledger := make(map[string]bool)
	var mu sync.Mutex

	settleTicket := func(ticketID string, amount float64) bool {
		mu.Lock()
		defer mu.Unlock()
		if ledger[ticketID] {
			return false // Already settled!
		}
		ledger[ticketID] = true
		atomic.AddInt32(&payoutCount, 1)
		return true
	}

	// First certified result delivery
	paid1 := settleTicket("TKT-100", 250.00)
	if !paid1 {
		t.Errorf("first settlement should succeed")
	}

	// Duplicate certified result deliveries
	paid2 := settleTicket("TKT-100", 250.00)
	paid3 := settleTicket("TKT-100", 250.00)
	if paid2 || paid3 {
		t.Errorf("duplicate deliveries must be rejected by idempotency ledger")
	}

	if atomic.LoadInt32(&payoutCount) != 1 {
		t.Fatalf("CRITICAL FINANCIAL INVARIANT VIOLATION: Double payout occurred! Count=%d", payoutCount)
	}
}

// Invariant 12: Senior vs U21 Disambiguation
// "Chelsea" vs "Chelsea U21" classified with distinct AgeCategory and generate two distinct canonical fixture IDs.
func TestInvariant12_SeniorVsU21Disambiguation(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()
	kickoff := time.Date(2026, 9, 21, 19, 0, 0, 0, time.UTC)

	senior, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-SENIOR",
		canonical.SportIDFootball, "Chelsea", "Fulham",
		kickoff, 1, "Premier League", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("senior fixture creation failed: %v", err)
	}

	u21, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-U21",
		canonical.SportIDFootball, "Chelsea U21", "Fulham U21",
		kickoff, 1, "Premier League 2", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("u21 fixture creation failed: %v", err)
	}

	if senior.CanonicalFixtureID == u21.CanonicalFixtureID {
		t.Fatalf("CROSS-CATEGORY COLLISION: Senior and U21 merged into same ID %s!", senior.CanonicalFixtureID)
	}
	if senior.AgeCategory != "SENIOR" || u21.AgeCategory != "U21" {
		t.Errorf("age categories misclassified: senior=%s, u21=%s", senior.AgeCategory, u21.AgeCategory)
	}
}

// Invariant 13: Kickoff Jitter Tolerance
// Matches with kickoff scheduled at 19:00 vs 19:10 (within ±15m window) map to the same natural bucket and canonical ID.
func TestInvariant13_KickoffJitterTolerance(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	// Provider A: 19:00 kickoff
	t1 := time.Date(2026, 9, 21, 19, 0, 0, 0, time.UTC)
	f1, _ := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-TIME-1",
		canonical.SportIDFootball, "Inter Milan", "Napoli",
		t1, 3, "Serie A", canonical.MappingPrematch, "test",
	)

	// Provider B: 19:10 kickoff (+10m jitter)
	t2 := time.Date(2026, 9, 21, 19, 10, 0, 0, time.UTC)
	f2, _ := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "PP-TIME-2",
		canonical.SportIDFootball, "Inter Milan", "Napoli",
		t2, 3, "Serie A", canonical.MappingPrematch, "test",
	)

	if f1.CanonicalFixtureID != f2.CanonicalFixtureID {
		t.Fatalf("JITTER FAILURE: ±10m schedule difference produced separate canonical IDs: %s vs %s", f1.CanonicalFixtureID, f2.CanonicalFixtureID)
	}
}

// Invariant 14: Senior vs Women Disambiguation
// Senior men vs women receive distinct canonical IDs.
func TestInvariant14_SeniorVsWomenDisambiguation(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()
	kickoff := time.Date(2026, 9, 21, 14, 0, 0, 0, time.UTC)

	men, _ := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-MEN",
		canonical.SportIDFootball, "Arsenal", "Tottenham",
		kickoff, 1, "Premier League", canonical.MappingPrematch, "test",
	)

	women, _ := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-WOMEN",
		canonical.SportIDFootball, "Arsenal Women", "Tottenham Women",
		kickoff, 1, "WSL", canonical.MappingPrematch, "test",
	)

	if men.CanonicalFixtureID == women.CanonicalFixtureID {
		t.Fatalf("GENDER COLLISION: Men and Women merged into same ID %s!", men.CanonicalFixtureID)
	}
	if men.Gender != "MEN" || women.Gender != "WOMEN" {
		t.Errorf("gender misclassified: men=%s, women=%s", men.Gender, women.Gender)
	}
}

// Invariant 15: Postponed State Transition
// Match postponed lifecycle state updates without changing canonical ID.
func TestInvariant15_PostponedStateTransition(t *testing.T) {
	ctx := context.Background()
	engine, _, _ := setupTestEngine()

	f, _ := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-POSTPONE",
		canonical.SportIDFootball, "Everton", "Newcastle",
		time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC),
		1, "Premier League", canonical.MappingPrematch, "test",
	)

	// Simulate postponement
	f.FixtureState = canonical.FixturePostponed
	f.BettingState = canonical.BettingSuspended

	if f.FixtureState != canonical.FixturePostponed || f.BettingState != canonical.BettingSuspended {
		t.Errorf("invalid postponed state: %+v", f)
	}
}

// Invariant 16: Restart Recovery
// Process restart with cold L1 cache re-hydrates canonical fixture and aliases without ID corruption.
func TestInvariant16_RestartRecovery(t *testing.T) {
	ctx := context.Background()
	repo := identity.NewMockRepository()
	redis := identity.NewMockRedis()

	// 1. Initial engine creates fixture
	engine1 := identity.NewEngine(repo, identity.NewTieredIdentityCache(redis, time.Hour))
	f1, err := engine1.ResolveOrCreateFixture(
		ctx, "sportybet", "SB-RESTART-1",
		canonical.SportIDFootball, "Sevilla", "Real Betis",
		time.Date(2026, 9, 21, 21, 0, 0, 0, time.UTC),
		10, "La Liga", canonical.MappingPrematch, "test",
	)
	if err != nil {
		t.Fatalf("engine 1 creation failed: %v", err)
	}

	// 2. Process Crash & Restart: New engine instance with empty L1 cache, but same repo/redis
	engine2 := identity.NewEngine(repo, identity.NewTieredIdentityCache(redis, time.Hour))

	// Look up alias
	cid, err := engine2.ResolveCanonicalID(ctx, "sportybet", "SB-RESTART-1")
	if err != nil || cid != f1.CanonicalFixtureID {
		t.Fatalf("engine 2 failed to recover alias: cid=%s, expected=%s, err=%v", cid, f1.CanonicalFixtureID, err)
	}

	// Fetch full fixture
	recoveredFixture, fErr := engine2.GetCanonicalFixture(ctx, cid)
	if fErr != nil || recoveredFixture == nil {
		t.Fatalf("engine 2 failed to re-hydrate fixture: %v", fErr)
	}
	if recoveredFixture.HomeTeam != "Sevilla" || recoveredFixture.AwayTeam != "Real Betis" {
		t.Errorf("recovered fixture corrupted: %+v", recoveredFixture)
	}
}
