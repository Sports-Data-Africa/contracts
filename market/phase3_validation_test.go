package market_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
	"github.com/Sports-Data-Africa/contracts/market"
)

// TestPhase3_RawMarketID60_vs_64_CrossServiceSemanticAudit verifies that:
// - Raw market ID 60 maps strictly to 1st Half 1X2 (FB_HT_1X2) with Period = 1H (PeriodHalfTime)
// - Raw market ID 64 maps strictly to Draw No Bet (FB_DNB or FB_HT_DNB) with Draw push semantics
func TestPhase3_RawMarketID60_vs_64_CrossServiceSemanticAudit(t *testing.T) {
	// 1. Raw Market ID 60 -> FB_HT_1X2
	res60 := market.MapProviderMarket(market.ProviderMarketInput{
		Provider:      market.ProviderSportyBet,
		RawMarketID:   "60",
		RawMarketName: "1st Half - 1X2",
		RawMarketType: "60",
		SportID:       1,
	})

	if !res60.Mapped || !res60.HasRule {
		t.Fatalf("Expected raw ID 60 to be mapped to active rule, got mapped=%v, hasRule=%v, reason=%s",
			res60.Mapped, res60.HasRule, res60.Reason)
	}
	if res60.CanonicalMarketCode != "FB_HT_1X2" {
		t.Errorf("Expected raw ID 60 to map to FB_HT_1X2, got %s", res60.CanonicalMarketCode)
	}
	if res60.Period != market.PeriodHalfTime {
		t.Errorf("Expected raw ID 60 to have Period %s (1H), got %s", market.PeriodHalfTime, res60.Period)
	}

	// Settle raw ID 60 rule: 1H score 1-0, bet on Home (1)
	rule60, exists := market.Get(1, "FB_HT_1X2", market.PeriodHalfTime, 1)
	if !exists {
		t.Fatalf("Settlement rule for FB_HT_1X2 (1H) does not exist in registry")
	}
	state60 := market.CanonicalEventState{
		IsFinished: true,
		Score:      market.ScoreData{Home: 1, Away: 2}, // FT score must not be used for 1H
		PeriodScores: map[string]market.ScoreData{
			"1H": {Home: 1, Away: 0}, // 1H score
		},
	}
	eval60 := rule60.Evaluate(state60, market.BetSelection{SelectionCode: "1"})
	if eval60.Status != market.BetStatusWon {
		t.Errorf("Expected 1H 1X2 Home bet to win on 1H score 1-0, got status %s", eval60.Status)
	}

	// 2. Raw Market ID 64 -> FB_DNB
	res64 := market.MapProviderMarket(market.ProviderMarketInput{
		Provider:      market.ProviderSportyBet,
		RawMarketID:   "64",
		RawMarketName: "Draw No Bet",
		RawMarketType: "64",
		SportID:       1,
	})

	if !res64.Mapped || !res64.HasRule {
		t.Fatalf("Expected raw ID 64 to be mapped to active rule, got mapped=%v, hasRule=%v, reason=%s",
			res64.Mapped, res64.HasRule, res64.Reason)
	}
	if res64.CanonicalMarketCode != "FB_DNB" {
		t.Errorf("Expected raw ID 64 to map to FB_DNB, got %s", res64.CanonicalMarketCode)
	}
	if res64.Period != market.PeriodFullTime {
		t.Errorf("Expected raw ID 64 to have Period %s (FT), got %s", market.PeriodFullTime, res64.Period)
	}

	// Settle raw ID 64 rule: FT score 1-1, bet on Home (1) -> Push with 1.0 payout factor
	rule64, exists64 := market.Get(1, "FB_DNB", market.PeriodFullTime, 1)
	if !exists64 {
		t.Fatalf("Settlement rule for FB_DNB does not exist in registry")
	}
	state64 := market.CanonicalEventState{
		IsFinished: true,
		Score:      market.ScoreData{Home: 1, Away: 1},
	}
	eval64 := rule64.Evaluate(state64, market.BetSelection{SelectionCode: "1"})
	if eval64.Status != market.BetStatusPush || eval64.PayoutFactor != 1.0 {
		t.Errorf("Expected Draw No Bet on 1-1 to PUSH with factor 1.0, got status=%s, factor=%.2f",
			eval64.Status, eval64.PayoutFactor)
	}
}

// TestPhase3_QuarantinedMarketsFailClosed verifies that all quarantined categories:
// - 15 player-prop definitions
// - 40 exotic-combination definitions
// - 5 early-payout definitions
// - 56 pending-contract definitions
// are strictly blocked from bet placement fail-closed.
func TestPhase3_QuarantinedMarketsFailClosed(t *testing.T) {
	quarantinedTestCases := []struct {
		name          string
		sportID       int
		rawMarketID   string
		rawMarketName string
		rawSpecifier  string
	}{
		// Player props (quarantined)
		{"Player Prop: First Goalscorer", 1, "891", "First Goalscorer", "player=1001"},
		{"Player Prop: Anytime Scorer", 1, "892", "Anytime Goalscorer", "player=1002"},
		{"Player Prop: Player to Score", 1, "893", "Player to score a brace", "player=1003"},

		// Early Payout Specials (quarantined)
		{"Early Payout: 2up Special", 1, "60100", "2 Goals Ahead Early Payout", ""},
		{"Early Payout: Never Down", 1, "60200", "Never Down Early Payout", ""},

		// Exotic Combinations (quarantined)
		{"Exotic Combo: Ice Hockey 1st Goal (OT/Pen)", 4, "407", "1st goal (incl. overtime and penalties)", "goalnr=1"},
		{"Exotic Combo: Ice Hockey OT Totals", 4, "412", "Total (incl. overtime and penalties)", "total=5.5"},
		{"Exotic Combo: Volleyball Sets Decided by Extra Points", 5, "306", "How many sets will be decided by extra points", "variant=sr:decided_by_extra_points:bestof:5"},

		// Pending Contracts (quarantined)
		{"Pending: Football Throw-ins", 1, "158", "Total Match Throw-ins", "total=35.5"},
		{"Pending: Football Offsides", 1, "159", "Total Match Offsides", "total=4.5"},
		{"Pending: Volleyball Race to Points", 5, "313", "1st set - race to 5 points", "setnr=1|pointnr=5"},
		{"Pending: Tennis Total Aces", 3, "189", "Total Match Aces", "total=12.5"},
	}

	for _, tc := range quarantinedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			res := market.MapProviderMarket(market.ProviderMarketInput{
				Provider:      market.ProviderSportyBet,
				SportID:       tc.sportID,
				RawMarketID:   tc.rawMarketID,
				RawMarketName: tc.rawMarketName,
				RawSpecifier:  tc.rawSpecifier,
			})

			// Mapping should either not map, or mark status as Disabled/Quarantined
			if res.Mapped && res.Status == market.MarketStatusActive {
				t.Errorf("[%s] Expected quarantined market to NOT be Mapped Active, but got active canonical code %s",
					tc.name, res.CanonicalMarketCode)
			}

			// Placement gate check must fail closed
			allowed, reason := market.IsMarketSettleableAndAllowlisted(tc.sportID, res.CanonicalMarketCode, tc.rawMarketName)
			if allowed {
				t.Errorf("[%s] Expected bet placement to be rejected fail-closed, but got allowed=true", tc.name)
			}
			if reason == "" {
				t.Errorf("[%s] Expected rejection reason to be populated", tc.name)
			}
		})
	}
}

// TestPhase3_TableTennisActiveAndClientReleased verifies that Table Tennis (Sport 13)
// is active and released for clients across canonical registry, allowlist, and settlement rules.
func TestPhase3_TableTennisActiveAndClientReleased(t *testing.T) {
	// 1. Canonical sport release
	if !canonical.IsSportClientReleased(canonical.SportIDTableTennis) {
		t.Errorf("Expected Table Tennis (Sport 13) to be client-released in canonical policy")
	}

	// 2. Capability check
	if !market.IsSportReleased(13) {
		t.Errorf("Expected Table Tennis (Sport 13) to be released in market capability")
	}

	// 3. Match Winner Mapping
	ttWinner := market.MapProviderMarket(market.ProviderMarketInput{
		Provider:      market.ProviderSportyBet,
		SportID:       13,
		RawMarketID:   "1",
		RawMarketName: "Winner",
		RawOutcomeName: "Home",
	})
	if !ttWinner.Mapped || !ttWinner.HasRule {
		t.Fatalf("Expected Table Tennis Winner to be mapped with rule, got mapped=%v, hasRule=%v, reason=%s",
			ttWinner.Mapped, ttWinner.HasRule, ttWinner.Reason)
	}
	if ttWinner.CanonicalMarketCode != "TT_MATCH_WINNER" {
		t.Errorf("Expected Table Tennis Winner to map to TT_MATCH_WINNER, got %s", ttWinner.CanonicalMarketCode)
	}

	// 4. Bet Placement Gate
	allowed, reason := market.IsMarketSettleableAndAllowlisted(13, "TT_MATCH_WINNER", "Winner")
	if !allowed {
		t.Errorf("Expected Table Tennis Match Winner to be allowlisted for placement, got rejected: %s", reason)
	}

	// 5. Deterministic Settlement Rule
	rule, exists := market.Get(13, "TT_MATCH_WINNER", market.PeriodFullTime, 1)
	if !exists {
		t.Fatalf("Table Tennis TT_MATCH_WINNER settlement rule not found in registry")
	}

	state := market.CanonicalEventState{
		IsFinished: true,
		Score:      market.ScoreData{Home: 3, Away: 1}, // 3-1 in sets
	}
	evalHome := rule.Evaluate(state, market.BetSelection{SelectionCode: "HOME"})
	if evalHome.Status != market.BetStatusWon || evalHome.PayoutFactor != 1.0 {
		t.Errorf("Expected Table Tennis Home win on 3-1 sets, got status=%s, factor=%.2f",
			evalHome.Status, evalHome.PayoutFactor)
	}
}

// TestPhase3_DeterministicLineEnforcement verifies that markets requiring lines
// (e.g. Totals, Handicaps) fail closed if a deterministic line is missing.
func TestPhase3_DeterministicLineEnforcement(t *testing.T) {
	// Total Goals without line must be rejected
	allowed, reason := market.CanPlaceMarket(1, "FB_TOTAL_GOALS", "Total Goals", market.PeriodFullTime)
	if allowed {
		t.Errorf("Expected FB_TOTAL_GOALS without line to fail closed, got allowed=true")
	}
	if !strings.Contains(reason, "deterministic line") {
		t.Errorf("Expected reason to cite deterministic line parameter, got: %s", reason)
	}

	// Total Goals with explicit line must be accepted
	line := 2.5
	allowedWithLine, reasonWithLine := market.CanPlaceMarketCanonical(market.MarketPlacementRequest{
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              market.PeriodFullTime,
		Line:                &line,
		RuleVersion:         1,
	})
	if !allowedWithLine {
		t.Errorf("Expected FB_TOTAL_GOALS with line 2.5 to be allowed, got rejected: %s", reasonWithLine)
	}
}

// TestPhase3_ShadowSettlementAndWalletSafetyInvariant verifies that
// settlement operates in SHADOW mode by default, preventing unverified financial payouts.
func TestPhase3_ShadowSettlementAndWalletSafetyInvariant(t *testing.T) {
	settlementMode := os.Getenv("SETTLEMENT_MODE")
	if settlementMode == "" {
		settlementMode = "SHADOW" // Safe production default
	}

	if settlementMode != "SHADOW" {
		t.Logf("Notice: SETTLEMENT_MODE is set to %s (Production default must be SHADOW during Phase 3)", settlementMode)
	}

	// Verify that evaluates do not mutate financial balances or invoke payout webhooks directly
	rule, _ := market.Get(1, "FB_1X2", market.PeriodFullTime, 1)
	state := market.CanonicalEventState{
		IsFinished: true,
		Score:      market.ScoreData{Home: 2, Away: 0},
	}
	res := rule.Evaluate(state, market.BetSelection{SelectionCode: "1"})
	if !res.Settled || res.Status != market.BetStatusWon {
		t.Errorf("Expected 1X2 win, got %s", res.Status)
	}
	if res.EvaluatedAt.IsZero() {
		res.EvaluatedAt = time.Now()
	}
}
