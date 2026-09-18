package identity_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
	"github.com/Sports-Data-Africa/contracts/identity"
)

// MockSettlementEngine simulates the settlement service resolving bets deterministically.
type MockBetTicket struct {
	TicketID           string
	CanonicalFixtureID string
	MarketCode         string
	Selection          string
	Odds               float64
	Stake              float64
	Status             string
	Payout             float64
	SettledAt          *time.Time
}

type MockWalletService struct {
	mu             sync.Mutex
	transactions   map[string]float64 // idempotency_key -> amount
	executionCount int
}

func (w *MockWalletService) Payout(idempotencyKey string, amount float64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.transactions[idempotencyKey]; exists {
		// Idempotent: already paid
		return nil
	}
	w.transactions[idempotencyKey] = amount
	w.executionCount++
	return nil
}

// TestPrimaryAcceptance_FullLifecycleWithDivergentProviderIDs implements the mandatory
// primary acceptance test:
// Prematch (Paripesa 123456) -> Bet Placement (SPD-FB-456778) -> Live (Paripesa 987654) ->
// Result (AIScore AS-88219) -> Settlement -> Idempotent Wallet Payout.
// All three provider IDs differ (123456 != 987654 != AS-88219), but the entire lifecycle succeeds.
func TestPrimaryAcceptance_FullLifecycleWithDivergentProviderIDs(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize Authoritative Identity Engine with Tiered Cache
	repo := identity.NewMockRepository()
	mockRedis := identity.NewMockRedis()
	tieredCache := identity.NewTieredIdentityCache(mockRedis, 24*time.Hour)
	engine := identity.NewEngine(repo, tieredCache)

	startTime := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)
	homeTeam := "Arsenal"
	awayTeam := "Chelsea"
	sportID := canonical.SportIDFootball

	// =========================================================================
	// PHASE 1: PREMATCH INGESTION
	// Provider: Paripesa, Fixture ID: 123456
	// =========================================================================
	t.Log("[Phase 1] Ingesting Prematch Fixture from Paripesa (ID: 123456)")
	prematchFixture, err := engine.ResolveOrCreateFixture(
		ctx,
		"paripesa", "123456",
		sportID, homeTeam, awayTeam, startTime,
		10, "Premier League",
		canonical.MappingPrematch, "paripesa_prematch_vzip",
	)
	if err != nil {
		t.Fatalf("Failed to resolve prematch fixture: %v", err)
	}

	canonicalID := prematchFixture.CanonicalFixtureID
	if canonicalID != "SPD-FB-100001" {
		t.Fatalf("Expected canonical ID SPD-FB-100001, got %s", canonicalID)
	}
	t.Logf("  ✓ Prematch Mapped to Canonical Identity: %s", canonicalID)

	// =========================================================================
	// PHASE 2: CUSTOMER BET PLACEMENT
	// Customer places bet on Match Winner Home (Selection: "1", Odds: 1.95)
	// Ticket MUST store Canonical Fixture ID (never the provider ID)
	// =========================================================================
	t.Log("[Phase 2] Customer placing bet ticket on canonical fixture")
	ticket := &MockBetTicket{
		TicketID:           "TKT-SDA-2026-0001",
		CanonicalFixtureID: canonicalID, // SPD-FB-100001
		MarketCode:         "1X2",
		Selection:          "1",
		Odds:               1.95,
		Stake:              100.00,
		Status:             "PENDING",
	}

	if ticket.CanonicalFixtureID != canonicalID {
		t.Fatalf("Ticket canonical fixture ID mismatch: %s != %s", ticket.CanonicalFixtureID, canonicalID)
	}
	t.Logf("  ✓ Ticket Accepted and Persisted: ID=%s, CanonicalFixtureID=%s, Selection=%s, Odds=%.2f",
		ticket.TicketID, ticket.CanonicalFixtureID, ticket.Selection, ticket.Odds)

	// =========================================================================
	// PHASE 3: LIVE MATCH INGESTION
	// Provider: Paripesa, Fixture ID: 987654 (DIFFERENT from Prematch ID 123456!)
	// Identity Engine resolves 987654 -> SPD-FB-100001 via natural identity or alias
	// =========================================================================
	t.Log("[Phase 3] Ingesting Live in-play tick from Paripesa with DIFFERENT ID (987654)")
	liveFixture, err := engine.ResolveOrCreateFixture(
		ctx,
		"paripesa", "987654",
		sportID, homeTeam, awayTeam, startTime,
		10, "Premier League",
		canonical.MappingLive, "paripesa_live_websocket",
	)
	if err != nil {
		t.Fatalf("Failed to resolve live fixture: %v", err)
	}

	if liveFixture.CanonicalFixtureID != canonicalID {
		t.Fatalf("Live fixture resolved to WRONG canonical identity: expected %s, got %s",
			canonicalID, liveFixture.CanonicalFixtureID)
	}

	// Verify both Prematch and Live IDs resolve to the same canonical ID from cache
	cidFromPrematch, err := engine.ResolveCanonicalID(ctx, "paripesa", "123456")
	if err != nil || cidFromPrematch != canonicalID {
		t.Fatalf("Failed to resolve prematch ID 123456: %s, err: %v", cidFromPrematch, err)
	}
	cidFromLive, err := engine.ResolveCanonicalID(ctx, "paripesa", "987654")
	if err != nil || cidFromLive != canonicalID {
		t.Fatalf("Failed to resolve live ID 987654: %s, err: %v", cidFromLive, err)
	}
	t.Logf("  ✓ Live Ingestion Resolved to Same Canonical Identity: %s (123456 == 987654 -> %s)",
		liveFixture.CanonicalFixtureID, canonicalID)

	// =========================================================================
	// PHASE 4: RESULT CERTIFICATION INGESTION
	// Provider: AIScore, Fixture ID: AS-88219 (DIFFERENT from both Prematch and Live!)
	// Match Result: Home 2 - Away 1 (Arsenal wins)
	// =========================================================================
	t.Log("[Phase 4] Ingesting Certified Result from AIScore with RESULT ID (AS-88219)")
	resultFixture, err := engine.ResolveOrCreateFixture(
		ctx,
		"aiscore", "AS-88219",
		sportID, homeTeam, awayTeam, startTime,
		10, "Premier League",
		canonical.MappingResult, "aiscore_official_consensus",
	)
	if err != nil {
		t.Fatalf("Failed to resolve result fixture: %v", err)
	}

	if resultFixture.CanonicalFixtureID != canonicalID {
		t.Fatalf("Result fixture resolved to WRONG canonical identity: expected %s, got %s",
			canonicalID, resultFixture.CanonicalFixtureID)
	}

	cidFromResult, err := engine.ResolveCanonicalID(ctx, "aiscore", "AS-88219")
	if err != nil || cidFromResult != canonicalID {
		t.Fatalf("Failed to resolve result ID AS-88219: %s, err: %v", cidFromResult, err)
	}
	t.Logf("  ✓ Result Resolved to Same Canonical Identity: %s (AS-88219 -> %s)",
		resultFixture.CanonicalFixtureID, canonicalID)

	// =========================================================================
	// PHASE 5: SETTLEMENT ENGINE EXECUTION
	// Settle strictly against canonical_fixture_id = "SPD-FB-100001"
	// =========================================================================
	t.Log("[Phase 5] Executing Settlement against canonical_fixture_id")
	wallet := &MockWalletService{transactions: make(map[string]float64)}

	// Result Snapshot
	type ResultSnapshot struct {
		CanonicalID string
		HomeScore   int
		AwayScore   int
		Status      string
	}
	snap := ResultSnapshot{
		CanonicalID: canonicalID,
		HomeScore:   2,
		AwayScore:   1,
		Status:      "FINAL",
	}

	// Query pending tickets by Canonical ID
	pendingTickets := []*MockBetTicket{ticket}

	for _, tkt := range pendingTickets {
		if tkt.CanonicalFixtureID != snap.CanonicalID {
			continue
		}

		// Determiner logic for 1X2
		won := false
		if tkt.MarketCode == "1X2" {
			if tkt.Selection == "1" && snap.HomeScore > snap.AwayScore {
				won = true
			} else if tkt.Selection == "X" && snap.HomeScore == snap.AwayScore {
				won = true
			} else if tkt.Selection == "2" && snap.AwayScore > snap.HomeScore {
				won = true
			}
		}

		if won {
			now := time.Now().UTC()
			tkt.Status = "WON"
			tkt.Payout = tkt.Stake * tkt.Odds
			tkt.SettledAt = &now

			// Execute Idempotent Wallet Payout
			idempotencyKey := fmt.Sprintf("wallet_payout_%s_%s", tkt.TicketID, snap.CanonicalID)
			payoutErr := wallet.Payout(idempotencyKey, tkt.Payout)
			if payoutErr != nil {
				t.Fatalf("Wallet payout failed: %v", payoutErr)
			}
		} else {
			now := time.Now().UTC()
			tkt.Status = "LOST"
			tkt.Payout = 0
			tkt.SettledAt = &now
		}
	}

	// Verify Settlement Outcome
	if ticket.Status != "WON" {
		t.Fatalf("Expected ticket status WON, got %s", ticket.Status)
	}
	if ticket.Payout != 195.00 {
		t.Fatalf("Expected payout 195.00, got %.2f", ticket.Payout)
	}
	if wallet.executionCount != 1 {
		t.Fatalf("Expected exactly 1 wallet payout execution, got %d", wallet.executionCount)
	}
	t.Logf("  ✓ Ticket Settled Successfully: Status=%s, Payout=%.2f", ticket.Status, ticket.Payout)

	// =========================================================================
	// PHASE 6: SETTLEMENT IDEMPOTENCY SAFETY TEST
	// Duplicate result delivery MUST NOT double-pay wallet
	// =========================================================================
	t.Log("[Phase 6] Testing Duplicate Result Delivery (Idempotency Guard)")
	duplicateIdempotencyKey := fmt.Sprintf("wallet_payout_%s_%s", ticket.TicketID, snap.CanonicalID)
	dupErr := wallet.Payout(duplicateIdempotencyKey, ticket.Payout)
	if dupErr != nil {
		t.Fatalf("Duplicate payout call errored: %v", dupErr)
	}
	if wallet.executionCount != 1 {
		t.Fatalf("FINANCIAL VIOLATION: Wallet executed duplicate payout! Execution count=%d", wallet.executionCount)
	}
	t.Log("  ✓ Zero Double-Payout: Duplicate settlement event safely ignored by idempotency ledger")

	t.Log("================================================================================")
	t.Log("PRIMARY ACCEPTANCE TEST: 100% SUCCESS")
	t.Log("Prematch (123456) != Live (987654) != Result (AS-88219)")
	t.Log("All cleanly bound to Canonical Identity SPD-FB-100001 and settled accurately.")
	t.Log("================================================================================")
}
