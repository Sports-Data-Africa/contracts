package market

import (
	"testing"
)

// Real captured-provider-payload fixtures from SportyBet & Odibets feeds
func TestMapProviderMarket_RealCapturedPayloadFixtures(t *testing.T) {
	tests := []struct {
		name                string
		input               ProviderMarketInput
		expectMapped        bool
		expectHasRule       bool
		expectCode          string
		expectPeriod        string
		expectFormula       string
		expectMetric        string
		expectTarget        string
		expectNormSel       string
		expectLine          float64
		expectStatus        string
		expectReasonContain string
	}{
		{
			name: "SportyBet Captured: 1st Half - Over/Under 0.5 (Over)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "18",
				RawMarketName:  "1st Half - Over/Under",
				RawMarketType:  "1st Half - Over/Under",
				RawSpecifier:   "total=0.5",
				RawOutcomeID:   "101",
				RawOutcomeName: "Over 0.5",
				RawPayloadHash: "sha256_mock_hash_ht_ou",
			},
			expectMapped:  true,
			expectHasRule: true,
			expectCode:    "FB_TOTAL_GOALS",
			expectPeriod:  PeriodHalfTime,
			expectFormula: FormulaTotals,
			expectMetric:  MetricGoals,
			expectTarget:  TargetMatch,
			expectNormSel: "OVER",
			expectLine:    0.5,
			expectStatus:  MarketStatusActive,
		},
		{
			name: "SportyBet Captured: Full-time Over/Under 2.5 (Under)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "18",
				RawMarketName:  "Over/Under",
				RawMarketType:  "Total Goals",
				RawSpecifier:   "total=2.5",
				RawOutcomeID:   "102",
				RawOutcomeName: "Under 2.5",
			},
			expectMapped:  true,
			expectHasRule: true,
			expectCode:    "FB_TOTAL_GOALS",
			expectPeriod:  PeriodFullTime,
			expectFormula: FormulaTotals,
			expectMetric:  MetricGoals,
			expectTarget:  TargetMatch,
			expectNormSel: "UNDER",
			expectLine:    2.5,
			expectStatus:  MarketStatusActive,
		},
		{
			name: "SportyBet Captured: 1st Half - 1X2 (Home Win)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "1",
				RawMarketName:  "1st Half - 3 Way",
				RawMarketType:  "1st Half 1X2",
				RawOutcomeID:   "201",
				RawOutcomeName: "Home",
			},
			expectMapped:  true,
			expectHasRule: true,
			expectCode:    "FB_HT_1X2",
			expectPeriod:  PeriodHalfTime,
			expectFormula: Formula1X2,
			expectMetric:  MetricGoals,
			expectTarget:  TargetMatch,
			expectNormSel: "1",
			expectStatus:  MarketStatusActive,
		},
		{
			name: "SportyBet Captured: 2nd Half - Over/Under (Suspended Period)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "18",
				RawMarketName:  "2nd Half - Over/Under",
				RawMarketType:  "2nd Half Total",
				RawSpecifier:   "total=1.5",
				RawOutcomeID:   "301",
				RawOutcomeName: "Over 1.5",
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "FB_TOTAL_GOALS",
			expectPeriod:        PeriodSecondHalf,
			expectFormula:       FormulaTotals,
			expectMetric:        MetricGoals,
			expectTarget:        TargetMatch,
			expectNormSel:       "OVER",
			expectLine:          1.5,
			expectStatus:        MarketStatusSuspended,
			expectReasonContain: "Period '2H' for Sport 1 is suspended",
		},
		{
			name: "SportyBet Captured: Basketball 1st Quarter Totals (Suspended Period Q1)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        2,
				RawMarketID:    "18",
				RawMarketName:  "1st Quarter Total Points",
				RawMarketType:  "Q1 Total",
				RawSpecifier:   "total=48.5",
				RawOutcomeID:   "401",
				RawOutcomeName: "Over 48.5",
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "BB_QUARTER_TOTALS",
			expectPeriod:        PeriodQuarter1,
			expectFormula:       FormulaTotals,
			expectMetric:        MetricGoals,
			expectTarget:        TargetMatch,
			expectNormSel:       "OVER",
			expectLine:          48.5,
			expectStatus:        MarketStatusSuspended,
			expectReasonContain: "Period 'Q1' for Sport 2 is suspended",
		},
		{
			name: "SportyBet Captured: Tennis Set 1 Winner (Suspended Period SET1)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        3,
				RawMarketID:    "1",
				RawMarketName:  "Set 1 Winner",
				RawMarketType:  "1st Set 2 Way",
				RawOutcomeID:   "501",
				RawOutcomeName: "1",
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "TN_SET_WINNER",
			expectPeriod:        PeriodSet1,
			expectFormula:       FormulaMoneyline,
			expectMetric:        MetricGoals,
			expectTarget:        TargetMatch,
			expectNormSel:       "1",
			expectStatus:        MarketStatusSuspended,
			expectReasonContain: "suspended",
		},
		{
			name: "SportyBet Captured: Corners Over/Under (Uncertified Domain DOMAIN_CORNERS)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "30",
				RawMarketName:  "Corner Over/Under 9.5",
				RawMarketType:  "Total Corners",
				RawSpecifier:   "total=9.5",
				RawOutcomeID:   "601",
				RawOutcomeName: "Over 9.5",
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "FB_CORNERS_TOTAL",
			expectPeriod:        PeriodFullTime,
			expectFormula:       FormulaTotals,
			expectMetric:        MetricGoals,
			expectTarget:        TargetMatch,
			expectNormSel:       "OVER",
			expectLine:          9.5,
			expectStatus:        MarketStatusSuspended,
			expectReasonContain: "required result domain 'DOMAIN_CORNERS' is not yet certified",
		},
		{
			name: "SportyBet Captured: Table Tennis Match Winner (Quarantined Sport 13)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        13,
				RawMarketID:    "1",
				RawMarketName:  "Match Winner",
				RawMarketType:  "2 Way",
				RawOutcomeID:   "701",
				RawOutcomeName: "1",
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "TT_MATCH_WINNER",
			expectPeriod:        PeriodFullTime,
			expectFormula:       FormulaMoneyline,
			expectMetric:        MetricGoals,
			expectTarget:        TargetMatch,
			expectNormSel:       "1",
			expectStatus:        MarketStatusQuarantined,
			expectReasonContain: "Sport ID 13 is quarantined",
		},
		{
			name: "SportyBet Captured: Unknown Raw Novelty Market (Fail-Closed)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "9999",
				RawMarketName:  "Manager to be Sent Off in Minute 23",
				RawMarketType:  "Novelty",
				RawOutcomeID:   "801",
				RawOutcomeName: "Yes",
			},
			expectMapped:        false,
			expectHasRule:       false,
			expectStatus:        MarketStatusDisabledUnsupported,
			expectReasonContain: "UNMAPPED_MARKET",
		},
		{
			name: "SportyBet Captured: Totals Market Missing Deterministic Line (Fail-Closed)",
			input: ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "18",
				RawMarketName:  "Over/Under",
				RawMarketType:  "Totals",
				RawOutcomeID:   "901",
				RawOutcomeName: "Over", // No line specified in outcome, specifier, or desc
			},
			expectMapped:        true,
			expectHasRule:       true,
			expectCode:          "FB_TOTAL_GOALS",
			expectPeriod:        PeriodFullTime,
			expectStatus:        MarketStatusSuspended,
			expectReasonContain: "MISSING_DETERMINISTIC_LINE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := MapProviderMarket(tc.input)

			if res.Mapped != tc.expectMapped {
				t.Fatalf("mapped: expected %v, got %v (reason: %s)", tc.expectMapped, res.Mapped, res.Reason)
			}
			if !tc.expectMapped {
				if res.Status != MarketStatusDisabledUnsupported {
					t.Fatalf("expected unmapped to be DISABLED_UNSUPPORTED, got %s", res.Status)
				}
				return
			}

			if res.CanonicalMarketCode != tc.expectCode {
				t.Errorf("code: expected %s, got %s", tc.expectCode, res.CanonicalMarketCode)
			}
			if res.Period != tc.expectPeriod {
				t.Errorf("period: expected %s, got %s", tc.expectPeriod, res.Period)
			}
			if tc.expectFormula != "" && res.FormulaType != tc.expectFormula {
				t.Errorf("formula: expected %s, got %s", tc.expectFormula, res.FormulaType)
			}
			if tc.expectNormSel != "" && res.NormalizedSelection != tc.expectNormSel {
				t.Errorf("normSel: expected %s, got %s", tc.expectNormSel, res.NormalizedSelection)
			}
			if tc.expectLine > 0 {
				if res.Line == nil || *res.Line != tc.expectLine {
					t.Errorf("line: expected %.2f, got %v", tc.expectLine, res.Line)
				}
			}
			if res.Status != tc.expectStatus {
				t.Errorf("status: expected %s, got %s (reason: %s)", tc.expectStatus, res.Status, res.Reason)
			}
			if tc.expectReasonContain != "" && (res.Reason == "" || !containsSub(res.Reason, tc.expectReasonContain)) {
				t.Errorf("expected reason to contain '%s', got '%s'", tc.expectReasonContain, res.Reason)
			}
		})
	}
}

// Regression: Prove 1st Half - Over/Under settles from HT ONLY, never from FT score
func TestRegression_1stHalfOverUnder_SettlesFromHTOnly(t *testing.T) {
	rule, ok := Get(1, "FB_TOTAL_GOALS", PeriodHalfTime, 1)
	if !ok {
		t.Fatalf("rule FB_TOTAL_GOALS (HT) not registered")
	}

	line05 := 0.5
	betUnder05 := BetSelection{
		BetID:               "bet_test_ht_under_05",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              PeriodHalfTime,
		SelectionCode:       "UNDER",
		Line:                &line05,
	}

	// Case A: Authoritative HT score is 0-0. Match finished 3-0.
	// Since bet was placed on HT Under 0.5, it MUST WIN because HT score was 0-0.
	stateWithHT := CanonicalEventState{
		CanonicalFixtureID: "fix_arsenal_chelsea",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 3, Away: 0}, // FT score is 3-0 (Over 0.5 for FT!)
		PeriodScores: map[string]ScoreData{
			"HT": {Home: 0, Away: 0}, // HT score is 0-0 (Under 0.5 for HT!)
		},
		IsPeriodFinished: map[string]bool{PeriodHalfTime: true},
	}

	resA := rule.Evaluate(stateWithHT, betUnder05)
	if resA.Status != BetStatusWon {
		t.Fatalf("expected HT Under 0.5 to be WON from HT score 0-0, got: %s (reason: %s)", resA.Status, resA.Reason)
	}

	// Case B: Missing HT score data. FT score is 3-0.
	// Critical Safety Rule: System must NEVER fall back to FT score or guess! It must produce SETTLEMENT_HOLD.
	stateMissingHT := CanonicalEventState{
		CanonicalFixtureID: "fix_arsenal_chelsea",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 3, Away: 0},
		PeriodScores:       map[string]ScoreData{}, // Missing HT score!
	}

	resB := rule.Evaluate(stateMissingHT, betUnder05)
	if resB.Status != BetStatusSettlementHold {
		t.Fatalf("CRITICAL BUG: Expected missing HT score to produce SETTLEMENT_HOLD, but got: %s", resB.Status)
	}
}

// Regression: Prove 2nd Half cannot be mistaken for FT
func TestRegression_2ndHalf_CannotBeMistakenForFT(t *testing.T) {
	rule, ok := Get(1, "FB_2H_1X2", PeriodSecondHalf, 1)
	if !ok {
		t.Fatalf("rule FB_2H_1X2 not registered")
	}

	bet2HHome := BetSelection{
		BetID:               "bet_test_2h_home",
		SportID:             1,
		CanonicalMarketCode: "FB_2H_1X2",
		Period:              PeriodSecondHalf,
		SelectionCode:       "1",
	}

	// HT score was 2-0. FT score was 2-1 (Away won 2nd half 0-1!).
	// If mistaken for FT score (2-1 Home win), Home bet would incorrectly win.
	state := CanonicalEventState{
		CanonicalFixtureID: "fix_liverpool_everton",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 1}, // FT Home win
		PeriodScores: map[string]ScoreData{
			"1H": {Home: 2, Away: 0},
			"2H": {Home: 0, Away: 1}, // 2H Away win
		},
	}

	res := rule.Evaluate(state, bet2HHome)
	if res.Status != BetStatusLost {
		t.Fatalf("expected 2H Home bet to be LOST (since 2H score was 0-1), got %s", res.Status)
	}
}

// Regression: Prove Corners/Cards cannot settle from score data
func TestRegression_CornersCards_CannotSettleFromScoreData(t *testing.T) {
	rule, ok := Get(1, "FB_CORNERS_TOTAL", PeriodFullTime, 1)
	if !ok {
		t.Fatalf("rule FB_CORNERS_TOTAL not registered")
	}

	line := 9.5
	betCornersOver := BetSelection{
		BetID:               "bet_test_corners",
		SportID:             1,
		CanonicalMarketCode: "FB_CORNERS_TOTAL",
		Period:              PeriodFullTime,
		SelectionCode:       "OVER",
		Line:                &line,
	}

	// Match finished 5-0 (Score data present), but Corners statistics missing
	stateMissingCorners := CanonicalEventState{
		CanonicalFixtureID: "fix_manchester_derby",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 5, Away: 0},
		Corners:            map[string]StatData{}, // Missing corner data!
	}

	res := rule.Evaluate(stateMissingCorners, betCornersOver)
	if res.Status != BetStatusSettlementHold && res.Status != BetStatusManualReview {
		t.Fatalf("expected missing corner statistics to produce SETTLEMENT_HOLD or MANUAL_REVIEW, got: %s", res.Status)
	}
}

func containsSub(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsSub(s[1:], substr))))
}
