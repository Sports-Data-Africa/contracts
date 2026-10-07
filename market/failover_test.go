package market

import (
	"testing"
	"time"
)

func TestProviderPrimaryAndFailover(t *testing.T) {
	now := time.Now()
	engine := NewFailoverEngine(30 * time.Second)

	sportyState := &CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100001",
		SportID:            1,
		Score:              ScoreData{Home: 2, Away: 1},
		Corners:            map[string]StatData{"FT": {Home: 6, Away: 4}},
	}
	odiState := &CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100001",
		SportID:            1,
		Score:              ScoreData{Home: 2, Away: 1},
		Corners:            map[string]StatData{"FT": {Home: 6, Away: 4}},
	}

	healthySporty := ProviderHealthState{
		Provider:      ProviderSportyBet,
		Connected:     true,
		LastMessageAt: now.Add(-5 * time.Second),
	}
	healthyOdi := ProviderHealthState{
		Provider:      ProviderOdibets,
		Connected:     true,
		LastMessageAt: now.Add(-5 * time.Second),
	}
	staleSporty := ProviderHealthState{
		Provider:      ProviderSportyBet,
		Connected:     true,
		LastMessageAt: now.Add(-60 * time.Second), // 60s ago > 30s threshold
	}

	// 1. Both healthy and agreeing -> BOTH_AGREED
	resBoth := engine.ResolveDomain(DomainScore, sportyState, odiState, healthySporty, healthyOdi, now)
	if !resBoth.Resolved || resBoth.HoldActive || resBoth.SourceProvider != "BOTH_AGREED" {
		t.Errorf("Expected BOTH_AGREED, got resolved=%v, hold=%v, src=%s", resBoth.Resolved, resBoth.HoldActive, resBoth.SourceProvider)
	}

	// 2. SportyBet stale -> Automatic failover to Odibets
	resFailover := engine.ResolveDomain(DomainScore, sportyState, odiState, staleSporty, healthyOdi, now)
	if !resFailover.Resolved || resFailover.HoldActive || resFailover.SourceProvider != ProviderOdibets {
		t.Errorf("Expected failover to ODIBETS, got resolved=%v, hold=%v, src=%s", resFailover.Resolved, resFailover.HoldActive, resFailover.SourceProvider)
	}

	// 3. SportyBet healthy alone (Odibets disconnected) -> SPORTYBET
	disconnectedOdi := ProviderHealthState{Provider: ProviderOdibets, Connected: false}
	resPrimary := engine.ResolveDomain(DomainScore, sportyState, nil, healthySporty, disconnectedOdi, now)
	if !resPrimary.Resolved || resPrimary.HoldActive || resPrimary.SourceProvider != ProviderSportyBet {
		t.Errorf("Expected primary SPORTYBET, got resolved=%v, hold=%v, src=%s", resPrimary.Resolved, resPrimary.HoldActive, resPrimary.SourceProvider)
	}
}

// TestDomainIsolationDiscrepancyHolding verifies that a disagreement in DOMAIN_CORNERS
// holds ONLY DOMAIN_CORNERS, while DOMAIN_SCORE continues to settle normally.
func TestDomainIsolationDiscrepancyHolding(t *testing.T) {
	now := time.Now()
	engine := NewFailoverEngine(30 * time.Second)

	sportyState := &CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100001",
		SportID:            1,
		Score:              ScoreData{Home: 2, Away: 1},                   // Score AGREE: 2-1
		Corners:            map[string]StatData{"FT": {Home: 6, Away: 4}}, // Corners DISAGREE: 6-4
	}
	odiState := &CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100001",
		SportID:            1,
		Score:              ScoreData{Home: 2, Away: 1},                   // Score AGREE: 2-1
		Corners:            map[string]StatData{"FT": {Home: 5, Away: 4}}, // Corners DISAGREE: 5-4
	}

	healthySporty := ProviderHealthState{
		Provider:      ProviderSportyBet,
		Connected:     true,
		LastMessageAt: now.Add(-5 * time.Second),
	}
	healthyOdi := ProviderHealthState{
		Provider:      ProviderOdibets,
		Connected:     true,
		LastMessageAt: now.Add(-5 * time.Second),
	}

	// Resolve DOMAIN_SCORE: Both agree on 2-1 -> Resolves normally!
	resScore := engine.ResolveDomain(DomainScore, sportyState, odiState, healthySporty, healthyOdi, now)
	if !resScore.Resolved || resScore.HoldActive {
		t.Errorf("DOMAIN_SCORE must NOT be held when scores agree, got resolved=%v, hold=%v", resScore.Resolved, resScore.HoldActive)
	}

	// Resolve DOMAIN_CORNERS: Material discrepancy (6-4 vs 5-4) -> ONLY corners are held!
	resCorners := engine.ResolveDomain(DomainCorners, sportyState, odiState, healthySporty, healthyOdi, now)
	if resCorners.Resolved || !resCorners.HoldActive {
		t.Errorf("DOMAIN_CORNERS MUST be HELD on discrepancy, got resolved=%v, hold=%v", resCorners.Resolved, resCorners.HoldActive)
	}
}
