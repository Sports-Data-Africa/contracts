package market

import (
	"strings"
	"testing"
	"time"
)

func TestPhase1ScoreDerivedSettlementRules(t *testing.T) {
	now := time.Now()

	// Finished match: FT 3-1, HT 2-0
	ft3_1ht2_0State := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-TEST-01",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 3, Away: 1}, // 2H was 1-1
		PeriodScores: map[string]ScoreData{
			"1H": {Home: 2, Away: 0},
			"HT": {Home: 2, Away: 0},
		},
		IsPeriodFinished: map[string]bool{PeriodHalfTime: true, PeriodFullTime: true},
		ScoreConfirmedAt: now,
	}

	tests := []struct {
		name          string
		state         CanonicalEventState
		bet           BetSelection
		expectStatus  string
		expectFactor  float64
		expectSettled bool
	}{
		// 1. FB_HT_BTTS: HT 2-0 -> BTTS NO Won, YES Lost
		{
			name:  "FB_HT_BTTS NO Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_BTTS",
				Period:              PeriodHalfTime,
				SelectionCode:       "NO",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},
		{
			name:  "FB_HT_BTTS YES Lost at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_BTTS",
				Period:              PeriodHalfTime,
				SelectionCode:       "YES",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusLost,
			expectFactor:  0.0,
			expectSettled: true,
		},

		// 2. FB_HT_DOUBLE_CHANCE: HT 2-0 -> 1X Won, 12 Won, X2 Lost
		{
			name:  "FB_HT_DOUBLE_CHANCE 1X Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_DOUBLE_CHANCE",
				Period:              PeriodHalfTime,
				SelectionCode:       "1X",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},
		{
			name:  "FB_HT_DOUBLE_CHANCE X2 Lost at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_DOUBLE_CHANCE",
				Period:              PeriodHalfTime,
				SelectionCode:       "X2",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusLost,
			expectFactor:  0.0,
			expectSettled: true,
		},

		// 3. FB_HT_DNB: HT 2-0 -> 1 Won, 2 Lost; Draw produces PUSH
		{
			name:  "FB_HT_DNB 1 Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_DNB",
				Period:              PeriodHalfTime,
				SelectionCode:       "1",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},
		{
			name: "FB_HT_DNB Push at HT 1-1",
			state: CanonicalEventState{
				CanonicalFixtureID: "SPD-FB-TEST-DNB-DRAW",
				SportID:            1,
				Status:             FixtureStatusHT,
				IsFinished:         false,
				IsPeriodFinished:   map[string]bool{PeriodHalfTime: true},
				PeriodScores:       map[string]ScoreData{"1H": {Home: 1, Away: 1}},
			},
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_DNB",
				Period:              PeriodHalfTime,
				SelectionCode:       "1",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusPush,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 4. FB_HT_ASIAN_HANDICAP: HT 2-0, Home (-0.5) Won
		{
			name:  "FB_HT_ASIAN_HANDICAP Home -0.5 Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_ASIAN_HANDICAP",
				Period:              PeriodHalfTime,
				SelectionCode:       "1",
				Line:                floatPtr(-0.5),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 5. FB_HT_EUROPEAN_HCP: HT 2-0, European Handicap (0:1) Home Won (2+0 > 1)
		{
			name:  "FB_HT_EUROPEAN_HCP Home Won at HT 2-0 with -1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_EUROPEAN_HCP",
				Period:              PeriodHalfTime,
				SelectionCode:       "1",
				Line:                floatPtr(-1.0),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 6. FB_HT_CORRECT_SCORE: HT 2-0
		{
			name:  "FB_HT_CORRECT_SCORE 2:0 Won",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_CORRECT_SCORE",
				Period:              PeriodHalfTime,
				SelectionCode:       "2:0",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 7. FB_HT_CLEAN_SHEET: HT 2-0 -> Home Clean Sheet Won, Away Lost
		{
			name:  "FB_HT_CLEAN_SHEET Home Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_CLEAN_SHEET",
				Period:              PeriodHalfTime,
				SelectionCode:       "HOME",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 8. FB_HT_WIN_TO_NIL: HT 2-0 -> Home Win To Nil Won
		{
			name:  "FB_HT_WIN_TO_NIL Home Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_WIN_TO_NIL",
				Period:              PeriodHalfTime,
				SelectionCode:       "HOME",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 9. FB_HOME_TOTALS_HT: HT Home 2 goals -> Over 1.5 Won
		{
			name:  "FB_HOME_TOTALS_HT Over 1.5 Won at HT Home=2",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HOME_TOTALS_HT",
				Period:              PeriodHalfTime,
				SelectionCode:       "OVER",
				Line:                floatPtr(1.5),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 10. FB_AWAY_TOTALS_HT: HT Away 0 goals -> Under 0.5 Won
		{
			name:  "FB_AWAY_TOTALS_HT Under 0.5 Won at HT Away=0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_AWAY_TOTALS_HT",
				Period:              PeriodHalfTime,
				SelectionCode:       "UNDER",
				Line:                floatPtr(0.5),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 11. FB_EXACT_GOALS: FT 3-1 = 4 goals
		{
			name:  "FB_EXACT_GOALS 4 Won at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_EXACT_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "4",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 12. FB_EXACT_GOALS_HT: HT 2-0 = 2 goals
		{
			name:  "FB_EXACT_GOALS_HT 2 Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_EXACT_GOALS_HT",
				Period:              PeriodHalfTime,
				SelectionCode:       "2",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 13. FB_GOAL_RANGE: FT 4 goals -> 4-6 Won, 2-3 Lost
		{
			name:  "FB_GOAL_RANGE 4-6 Won at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_GOAL_RANGE",
				Period:              PeriodFullTime,
				SelectionCode:       "4-6",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 14. FB_GOAL_RANGE_HT: HT 2 goals -> 2+ Won
		{
			name:  "FB_GOAL_RANGE_HT 2+ Won at HT 2-0",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_GOAL_RANGE_HT",
				Period:              PeriodHalfTime,
				SelectionCode:       "2+",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 15. FB_HOME_EXACT_GOALS: FT Home = 3
		{
			name:  "FB_HOME_EXACT_GOALS 3 Won at Home=3",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HOME_EXACT_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "3",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 16. FB_AWAY_EXACT_GOALS: FT Away = 1
		{
			name:  "FB_AWAY_EXACT_GOALS 1 Won at Away=1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_AWAY_EXACT_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "1",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 17. FB_HOME_ODD_EVEN: FT Home 3 -> ODD Won
		{
			name:  "FB_HOME_ODD_EVEN ODD Won at Home=3",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HOME_ODD_EVEN",
				Period:              PeriodFullTime,
				SelectionCode:       "ODD",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 18. FB_AWAY_ODD_EVEN: FT Away 1 -> ODD Won
		{
			name:  "FB_AWAY_ODD_EVEN ODD Won at Away=1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_AWAY_ODD_EVEN",
				Period:              PeriodFullTime,
				SelectionCode:       "ODD",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 19. FB_HT_ODD_EVEN: HT total 2 -> EVEN Won
		{
			name:  "FB_HT_ODD_EVEN EVEN Won at HT total=2",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_ODD_EVEN",
				Period:              PeriodHalfTime,
				SelectionCode:       "EVEN",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 20. FB_HIGHEST_SCORING_HALF: 1H had 2 goals, 2H had 2 goals (3-1 vs 2-0 -> 2H was 1-1 = 2 goals) -> EQUAL Won!
		{
			name:  "FB_HIGHEST_SCORING_HALF EQUAL Won when 1H=2 and 2H=2",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HIGHEST_SCORING_HALF",
				Period:              PeriodFullTime,
				SelectionCode:       "EQUAL",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 21. FB_HOME_HIGHEST_HALF: Home 1H=2, Home 2H=1 -> 1ST Won
		{
			name:  "FB_HOME_HIGHEST_HALF 1ST Won when Home 1H=2, 2H=1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HOME_HIGHEST_HALF",
				Period:              PeriodFullTime,
				SelectionCode:       "1ST",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 22. FB_AWAY_HIGHEST_HALF: Away 1H=0, Away 2H=1 -> 2ND Won
		{
			name:  "FB_AWAY_HIGHEST_HALF 2ND Won when Away 1H=0, 2H=1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_AWAY_HIGHEST_HALF",
				Period:              PeriodFullTime,
				SelectionCode:       "2ND",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 23. FB_WIN_BOTH_HALVES: 1H Home 2-0 (Home won), 2H 1-1 (Draw) -> Home did NOT win 2H -> Lost
		{
			name:  "FB_WIN_BOTH_HALVES HOME Lost when 2H is Draw",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_WIN_BOTH_HALVES",
				Period:              PeriodFullTime,
				SelectionCode:       "HOME",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusLost,
			expectFactor:  0.0,
			expectSettled: true,
		},

		// 24. FB_SCORE_BOTH_HALVES: Home scored 1H (2) and 2H (1) -> HOME Won
		{
			name:  "FB_SCORE_BOTH_HALVES HOME Won when Home scores in both halves",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_SCORE_BOTH_HALVES",
				Period:              PeriodFullTime,
				SelectionCode:       "HOME",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 25. FB_TO_WIN_EITHER_HALF: Home won 1H (2-0) -> HOME Won
		{
			name:  "FB_TO_WIN_EITHER_HALF HOME Won when Home wins 1H",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_TO_WIN_EITHER_HALF",
				Period:              PeriodFullTime,
				SelectionCode:       "HOME",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 26. FB_MULTI_GOALS: FT 3-1 = 4 goals -> 2-5 Won, 1-3 Lost
		{
			name:  "FB_MULTI_GOALS 2-5 Won at FT 4 goals",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_MULTI_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "2-5",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},
		{
			name:  "FB_MULTI_GOALS 1-3 Lost at FT 4 goals",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_MULTI_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "1-3",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusLost,
			expectFactor:  0.0,
			expectSettled: true,
		},

		// 27. FB_RESULT_BTTS: FT 3-1 (Home win & BTTS Yes) -> 1 & YES Won
		{
			name:  "FB_RESULT_BTTS 1 & YES Won at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_RESULT_BTTS",
				Period:              PeriodFullTime,
				SelectionCode:       "1 & YES",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},
		{
			name:  "FB_RESULT_BTTS 1 & NO Lost at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_RESULT_BTTS",
				Period:              PeriodFullTime,
				SelectionCode:       "1 & NO",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusLost,
			expectFactor:  0.0,
			expectSettled: true,
		},

		// 28. FB_RESULT_TOTALS: FT 3-1 (Home win & Over 2.5) -> 1 & OVER Won
		{
			name:  "FB_RESULT_TOTALS 1 & OVER 2.5 Won at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_RESULT_TOTALS",
				Period:              PeriodFullTime,
				SelectionCode:       "1 & OVER",
				Line:                floatPtr(2.5),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// 29. FB_BTTS_TOTALS: FT 3-1 (BTTS Yes & Over 2.5) -> YES & OVER Won
		{
			name:  "FB_BTTS_TOTALS YES & OVER 2.5 Won at FT 3-1",
			state: ft3_1ht2_0State,
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_BTTS_TOTALS",
				Period:              PeriodFullTime,
				SelectionCode:       "YES & OVER",
				Line:                floatPtr(2.5),
				RuleVersion:         1,
			},
			expectStatus:  BetStatusWon,
			expectFactor:  1.0,
			expectSettled: true,
		},

		// Edge case: Fail-Closed on missing 1H score for 1H-dependent market
		{
			name: "Fail-Closed: Missing 1H score holds settlement",
			state: CanonicalEventState{
				CanonicalFixtureID: "SPD-FB-NO-HT-SCORE",
				SportID:            1,
				Status:             FixtureStatusFT,
				IsFinished:         true,
				Score:              ScoreData{Home: 2, Away: 1},
				PeriodScores:       map[string]ScoreData{}, // Empty HT score!
			},
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_HT_BTTS",
				Period:              PeriodHalfTime,
				SelectionCode:       "YES",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusSettlementHold,
			expectFactor:  0.0,
			expectSettled: false,
		},

		// Edge case: Unfinished match returns BetStatusOpen
		{
			name: "Unfinished match returns BetStatusOpen for EventEnd market",
			state: CanonicalEventState{
				CanonicalFixtureID: "SPD-FB-LIVE",
				SportID:            1,
				Status:             FixtureStatusLive,
				IsFinished:         false,
				Score:              ScoreData{Home: 1, Away: 0},
			},
			bet: BetSelection{
				SportID:             1,
				CanonicalMarketCode: "FB_EXACT_GOALS",
				Period:              PeriodFullTime,
				SelectionCode:       "2",
				RuleVersion:         1,
			},
			expectStatus:  BetStatusOpen,
			expectFactor:  0.0,
			expectSettled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Evaluate(tc.state, tc.bet)
			if err != nil {
				t.Fatalf("Evaluate returned unexpected error: %v", err)
			}
			if res.Status != tc.expectStatus {
				t.Errorf("expected status %s, got %s (reason: %s)", tc.expectStatus, res.Status, res.Reason)
			}
			if res.Settled != tc.expectSettled {
				t.Errorf("expected settled %v, got %v", tc.expectSettled, res.Settled)
			}
			if res.PayoutFactor != tc.expectFactor {
				t.Errorf("expected factor %.2f, got %.2f", tc.expectFactor, res.PayoutFactor)
			}
		})
	}
}

func TestPhase1ProviderMappingCoverage(t *testing.T) {
	// Verify that incoming raw SportyBet market payloads map cleanly to the 29 new contracts
	testMappings := []struct {
		desc           string
		rawMarketName  string
		rawMarketType  string
		specifier      string
		rawOutcomeName string
		line           *float64
		expectCode     string
		expectPeriod   string
		expectActive   bool
	}{
		{desc: "1st Half - BTTS", rawMarketName: "1st Half - Both Teams to Score", rawOutcomeName: "Yes", expectCode: "FB_HT_BTTS", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Double Chance", rawMarketName: "1st Half - Double Chance", rawOutcomeName: "1X", expectCode: "FB_HT_DOUBLE_CHANCE", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Draw No Bet", rawMarketName: "1st Half - Draw No Bet", rawOutcomeName: "1", expectCode: "FB_HT_DNB", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Asian Handicap", rawMarketName: "1st Half - Asian Handicap", specifier: "hcp=-0.5", rawOutcomeName: "1", line: floatPtr(-0.5), expectCode: "FB_HT_ASIAN_HANDICAP", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - European Handicap", rawMarketName: "1st Half - Handicap", specifier: "hcp=-1", rawOutcomeName: "1", line: floatPtr(-1.0), expectCode: "FB_HT_EUROPEAN_HCP", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Correct Score", rawMarketName: "1st Half - Correct Score", rawOutcomeName: "1:0", expectCode: "FB_HT_CORRECT_SCORE", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Clean Sheet", rawMarketName: "1st Half - Clean Sheet", rawOutcomeName: "Home", expectCode: "FB_HT_CLEAN_SHEET", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Win to Nil", rawMarketName: "1st Half - Win to Nil", rawOutcomeName: "Home", expectCode: "FB_HT_WIN_TO_NIL", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Home Total Goals", rawMarketName: "1st Half - Home Total Goals", specifier: "total=0.5", rawOutcomeName: "Over 0.5", line: floatPtr(0.5), expectCode: "FB_HOME_TOTALS_HT", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "1st Half - Away Total Goals", rawMarketName: "1st Half - Away Total Goals", specifier: "total=0.5", rawOutcomeName: "Under 0.5", line: floatPtr(0.5), expectCode: "FB_AWAY_TOTALS_HT", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "Exact Goals", rawMarketName: "Exact Goals", rawOutcomeName: "2", expectCode: "FB_EXACT_GOALS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "1st Half - Exact Goals", rawMarketName: "1st Half - Exact Goals", rawOutcomeName: "1", expectCode: "FB_EXACT_GOALS_HT", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "Goal Range", rawMarketName: "Goal Range", rawOutcomeName: "2-3", expectCode: "FB_GOAL_RANGE", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "1st Half - Goal Range", rawMarketName: "1st Half - Goal Range", rawOutcomeName: "0-1", expectCode: "FB_GOAL_RANGE_HT", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "Home Exact Goals", rawMarketName: "Home Exact Goals", rawOutcomeName: "1", expectCode: "FB_HOME_EXACT_GOALS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Away Exact Goals", rawMarketName: "Away Exact Goals", rawOutcomeName: "0", expectCode: "FB_AWAY_EXACT_GOALS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Home Odd/Even", rawMarketName: "Home Odd/Even", rawOutcomeName: "Odd", expectCode: "FB_HOME_ODD_EVEN", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Away Odd/Even", rawMarketName: "Away Odd/Even", rawOutcomeName: "Even", expectCode: "FB_AWAY_ODD_EVEN", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "1st Half - Odd/Even", rawMarketName: "1st Half - Odd/Even", rawOutcomeName: "Odd", expectCode: "FB_HT_ODD_EVEN", expectPeriod: PeriodHalfTime, expectActive: true},
		{desc: "Highest Scoring Half", rawMarketName: "Highest Scoring Half", rawOutcomeName: "2nd", expectCode: "FB_HIGHEST_SCORING_HALF", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Home Highest Scoring Half", rawMarketName: "Home Highest Scoring Half", rawOutcomeName: "1st", expectCode: "FB_HOME_HIGHEST_HALF", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Away Highest Scoring Half", rawMarketName: "Away Highest Scoring Half", rawOutcomeName: "2nd", expectCode: "FB_AWAY_HIGHEST_HALF", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Win Both Halves", rawMarketName: "Win Both Halves", rawOutcomeName: "Home", expectCode: "FB_WIN_BOTH_HALVES", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Score in Both Halves", rawMarketName: "Score in Both Halves", rawOutcomeName: "Home", expectCode: "FB_SCORE_BOTH_HALVES", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "To Win Either Half", rawMarketName: "To Win Either Half", rawOutcomeName: "Home", expectCode: "FB_TO_WIN_EITHER_HALF", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Multi Goals", rawMarketName: "Multi Goals", rawOutcomeName: "2-4", expectCode: "FB_MULTI_GOALS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "1X2 and Both Teams to Score", rawMarketName: "1X2 & Both Teams to Score", rawOutcomeName: "1 & Yes", expectCode: "FB_RESULT_BTTS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "1X2 and Over/Under", rawMarketName: "1X2 & Over/Under 2.5", specifier: "total=2.5", rawOutcomeName: "1 & Over", line: floatPtr(2.5), expectCode: "FB_RESULT_TOTALS", expectPeriod: PeriodFullTime, expectActive: true},
		{desc: "Both Teams to Score and Over/Under", rawMarketName: "Both Teams to Score & Over/Under 2.5", specifier: "total=2.5", rawOutcomeName: "Yes & Over", line: floatPtr(2.5), expectCode: "FB_BTTS_TOTALS", expectPeriod: PeriodFullTime, expectActive: true},
	}

	for _, tm := range testMappings {
		t.Run(tm.desc, func(t *testing.T) {
			res := MapProviderMarket(ProviderMarketInput{
				Provider:       ProviderSportyBet,
				SportID:        1,
				RawMarketName:  tm.rawMarketName,
				RawMarketType:  tm.rawMarketType,
				RawSpecifier:   tm.specifier,
				RawOutcomeName: tm.rawOutcomeName,
				Line:           tm.line,
			})

			if !res.Mapped {
				t.Fatalf("expected market to be mapped, got unmapped: %s", res.Reason)
			}
			if !res.HasRule {
				t.Fatalf("expected market to have rule registered, but none found: %s", res.Reason)
			}
			if res.CanonicalMarketCode != tm.expectCode {
				t.Errorf("expected canonical code %s, got %s", tm.expectCode, res.CanonicalMarketCode)
			}
			if res.Period != tm.expectPeriod {
				t.Errorf("expected period %s, got %s", tm.expectPeriod, res.Period)
			}
			if tm.expectActive && res.Status != MarketStatusActive {
				t.Errorf("expected ACTIVE status, got %s (reason: %s)", res.Status, res.Reason)
			}
		})
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestPhase1EdgeCasesAndDefects(t *testing.T) {
	now := time.Now()

	validFTState := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-AUDIT-VALID",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 1},
		PeriodScores: map[string]ScoreData{
			"1H": {Home: 1, Away: 0},
			"HT": {Home: 1, Away: 0},
		},
		IsPeriodFinished: map[string]bool{PeriodHalfTime: true, PeriodFullTime: true},
		ScoreConfirmedAt: now,
	}

	inconsistentFTState := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-AUDIT-INCONSISTENT",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: 1, Away: 1}, // FT Home is 1, but HT Home is 2! Inconsistent!
		PeriodScores: map[string]ScoreData{
			"1H": {Home: 2, Away: 0},
			"HT": {Home: 2, Away: 0},
		},
		IsPeriodFinished: map[string]bool{PeriodHalfTime: true, PeriodFullTime: true},
		ScoreConfirmedAt: now,
	}

	negativeScoreState := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-AUDIT-NEGATIVE",
		SportID:            1,
		Status:             FixtureStatusFT,
		IsFinished:         true,
		Score:              ScoreData{Home: -1, Away: 2},
		PeriodScores: map[string]ScoreData{
			"1H": {Home: 0, Away: 1},
			"HT": {Home: 0, Away: 1},
		},
		IsPeriodFinished: map[string]bool{PeriodHalfTime: true, PeriodFullTime: true},
		ScoreConfirmedAt: now,
	}

	edgeTests := []struct {
		name         string
		state        CanonicalEventState
		bet          BetSelection
		expectStatus string
	}{
		// 1. Missing Line Specifier -> Hold
		{
			name:  "FB_TOTAL_GOALS missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_TOTAL_GOALS", Period: PeriodFullTime,
				SelectionCode: "OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HOME_TOTALS missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HOME_TOTALS", Period: PeriodFullTime,
				SelectionCode: "OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_AWAY_TOTALS missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_AWAY_TOTALS", Period: PeriodFullTime,
				SelectionCode: "OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HOME_TOTALS_HT missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HOME_TOTALS_HT", Period: PeriodHalfTime,
				SelectionCode: "OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_AWAY_TOTALS_HT missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_AWAY_TOTALS_HT", Period: PeriodHalfTime,
				SelectionCode: "OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_EUROPEAN_HANDICAP missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_EUROPEAN_HANDICAP", Period: PeriodFullTime,
				SelectionCode: "1", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HT_EUROPEAN_HCP missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HT_EUROPEAN_HCP", Period: PeriodHalfTime,
				SelectionCode: "1", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_ASIAN_HANDICAP missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_ASIAN_HANDICAP", Period: PeriodFullTime,
				SelectionCode: "1", Line: nil, LineQuarters: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HT_ASIAN_HANDICAP missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HT_ASIAN_HANDICAP", Period: PeriodHalfTime,
				SelectionCode: "1", Line: nil, LineQuarters: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_RESULT_TOTALS missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_RESULT_TOTALS", Period: PeriodFullTime,
				SelectionCode: "1 & OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_BTTS_TOTALS missing line holds",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_BTTS_TOTALS", Period: PeriodFullTime,
				SelectionCode: "YES & OVER", Line: nil, RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},

		// 2. Score Inconsistency (FT < HT) -> Hold
		{
			name:  "FB_WIN_BOTH_HALVES holds on FT < HT inconsistency",
			state: inconsistentFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_WIN_BOTH_HALVES", Period: PeriodFullTime,
				SelectionCode: "HOME", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_SCORE_BOTH_HALVES holds on FT < HT inconsistency",
			state: inconsistentFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_SCORE_BOTH_HALVES", Period: PeriodFullTime,
				SelectionCode: "HOME", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_TO_WIN_EITHER_HALF holds on FT < HT inconsistency",
			state: inconsistentFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_TO_WIN_EITHER_HALF", Period: PeriodFullTime,
				SelectionCode: "HOME", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HIGHEST_SCORING_HALF holds on FT < HT inconsistency",
			state: inconsistentFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HIGHEST_SCORING_HALF", Period: PeriodFullTime,
				SelectionCode: "1ST", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},

		// 3. Negative Score -> Hold
		{
			name:  "FB_1X2 holds on negative score",
			state: negativeScoreState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_1X2", Period: PeriodFullTime,
				SelectionCode: "1", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_TOTAL_GOALS holds on negative score",
			state: negativeScoreState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_TOTAL_GOALS", Period: PeriodFullTime,
				SelectionCode: "OVER", Line: floatPtr(2.5), RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_MULTI_GOALS holds on negative score",
			state: negativeScoreState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_MULTI_GOALS", Period: PeriodFullTime,
				SelectionCode: "2-4", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},

		// 4. Invalid Selection Codes -> Hold (NOT silent loss or wrong side)
		{
			name:  "FB_CLEAN_SHEET holds on invalid selection code",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_CLEAN_SHEET", Period: PeriodFullTime,
				SelectionCode: "FOOBAR", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HT_CLEAN_SHEET holds on invalid selection code",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HT_CLEAN_SHEET", Period: PeriodHalfTime,
				SelectionCode: "XYZ", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_WIN_TO_NIL holds on invalid selection code",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_WIN_TO_NIL", Period: PeriodFullTime,
				SelectionCode: "NONSENSE", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_HT_WIN_TO_NIL holds on invalid selection code",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_HT_WIN_TO_NIL", Period: PeriodHalfTime,
				SelectionCode: "UNKNOWN", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_MULTI_GOALS holds on invalid format (not hyphenated)",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_MULTI_GOALS", Period: PeriodFullTime,
				SelectionCode: "TWO_TO_FOUR", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_MULTI_GOALS holds on invalid inverted range (min > max)",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_MULTI_GOALS", Period: PeriodFullTime,
				SelectionCode: "5-2", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_RESULT_BTTS holds on invalid combo selection format",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_RESULT_BTTS", Period: PeriodFullTime,
				SelectionCode: "HOME_AND_YES_WITHOUT_AMPERSAND", RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_RESULT_TOTALS holds on invalid combo selection format",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_RESULT_TOTALS", Period: PeriodFullTime,
				SelectionCode: "INVALID_COMBO", Line: floatPtr(2.5), RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
		{
			name:  "FB_BTTS_TOTALS holds on invalid combo selection format",
			state: validFTState,
			bet: BetSelection{
				SportID: 1, CanonicalMarketCode: "FB_BTTS_TOTALS", Period: PeriodFullTime,
				SelectionCode: "INVALID_COMBO", Line: floatPtr(2.5), RuleVersion: 1,
			},
			expectStatus: BetStatusSettlementHold,
		},
	}

	for _, tc := range edgeTests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Evaluate(tc.state, tc.bet)
			if err != nil {
				t.Fatalf("Evaluate returned unexpected error: %v", err)
			}
			if res.Status != tc.expectStatus {
				t.Errorf("expected status %s, got %s (reason: %s)", tc.expectStatus, res.Status, res.Reason)
			}
		})
	}
}

func TestPhase2SecondHalfTelemetryAndFaultInjection(t *testing.T) {
	now := time.Now()

	// 1. Live match: in-play 2H with cumulative score (Score: 2-1, Period: 2H, IsFinished: false)
	// Must return BetStatusOpen (unresolved), NOT BetStatusWon / Lost, and Settled: false
	t.Run("Live 2H match with cumulative score remains OPEN and does not settle", func(t *testing.T) {
		liveState := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-LIVE",
			SportID:            1,
			Status:             FixtureStatusLive2H,
			IsFinished:         false,
			Score:              ScoreData{Home: 2, Away: 1},
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 1, Away: 0},
				"HT": {Home: 1, Away: 0},
			},
			ScoreConfirmedAt: now,
		}
		bet := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		res, err := Evaluate(liveState, bet)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusOpen {
			t.Errorf("expected BetStatusOpen for live 2H match, got %s", res.Status)
		}
		if res.Settled {
			t.Errorf("expected Settled=false for live 2H match, got true")
		}
	})

	// 2. Finished match with missing HT score: must fail closed to BetStatusSettlementHold
	t.Run("Finished match with missing HT score produces SETTLEMENT_HOLD", func(t *testing.T) {
		stateNoHT := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-NO-HT",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 3, Away: 1},
			PeriodScores:       map[string]ScoreData{}, // No 1H or HT!
			ScoreConfirmedAt:   now,
		}
		bet := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		res, err := Evaluate(stateNoHT, bet)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold when HT score is missing, got %s", res.Status)
		}
		if res.Settled {
			t.Errorf("expected Settled=false on missing HT score, got true")
		}
	})

	// 3. Finished match with contradictory scores (FT < HT): must fail closed to BetStatusSettlementHold
	t.Run("Contradictory FT and HT scores (FT < HT) produces SETTLEMENT_HOLD", func(t *testing.T) {
		stateContradictory := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-CONTRADICTORY",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 1, Away: 1}, // FT Home is 1, but HT Home was 2!
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 2, Away: 0},
				"HT": {Home: 2, Away: 0},
			},
			ScoreConfirmedAt: now,
		}
		bet := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		res, err := Evaluate(stateContradictory, bet)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold on FT < HT contradiction, got %s", res.Status)
		}
		if res.Settled {
			t.Errorf("expected Settled=false on contradiction, got true")
		}
	})

	// 4. Negative score telemetry: must fail closed to BetStatusSettlementHold
	t.Run("Negative score telemetry produces SETTLEMENT_HOLD", func(t *testing.T) {
		stateNegative := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-NEGATIVE",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: -1, Away: 2},
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 0, Away: 1},
			},
			ScoreConfirmedAt: now,
		}
		bet := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		res, err := Evaluate(stateNegative, bet)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold on negative score, got %s", res.Status)
		}
	})

	// 5. Invalid selection code on FB_2H_1X2: must fail closed to BetStatusSettlementHold
	t.Run("Invalid selection code on FB_2H_1X2 produces SETTLEMENT_HOLD", func(t *testing.T) {
		stateValid := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-VALID",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 2, Away: 1},
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 1, Away: 0},
			},
			ScoreConfirmedAt: now,
		}
		bet := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "INVALID_CODE",
			RuleVersion:         1,
		}
		res, err := Evaluate(stateValid, bet)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold on invalid selection code, got %s", res.Status)
		}
	})

	// 6. Valid post-match derivation: FT 2-1 with HT 1-0 -> 2H was 1-1 (Draw).
	// Selection "X" (Draw) MUST WIN. Selection "1" MUST LOSE.
	t.Run("Valid post-match derivation: FT 2-1, HT 1-0 -> 2H was 1-1 Draw", func(t *testing.T) {
		stateValid := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-VALID",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 2, Away: 1}, // 2H home: 2-1=1, away: 1-0=1 -> 1-1 Draw
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 1, Away: 0},
			},
			ScoreConfirmedAt: now,
		}
		betDraw := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "X",
			RuleVersion:         1,
		}
		resDraw, err := Evaluate(stateValid, betDraw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resDraw.Status != BetStatusWon || resDraw.PayoutFactor != 1.0 {
			t.Errorf("expected 2H Draw bet to WIN, got %s (factor: %.2f)", resDraw.Status, resDraw.PayoutFactor)
		}

		betHome := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		resHome, err := Evaluate(stateValid, betHome)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resHome.Status != BetStatusLost || resHome.PayoutFactor != 0.0 {
			t.Errorf("expected 2H Home bet to LOSE (since 2H was 1-1), got %s (factor: %.2f)", resHome.Status, resHome.PayoutFactor)
		}
	})

	// 7. Native 2H score present in PeriodScores: evaluates directly from PeriodScores["2H"]
	t.Run("Native 2H score in PeriodScores is evaluated directly", func(t *testing.T) {
		stateNative := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-NATIVE",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 3, Away: 1},
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 3, Away: 0},
				"2H": {Home: 0, Away: 1}, // 2H Away won 0-1
			},
			ScoreConfirmedAt: now,
		}
		betAway := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "2",
			RuleVersion:         1,
		}
		resAway, err := Evaluate(stateNative, betAway)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resAway.Status != BetStatusWon || resAway.PayoutFactor != 1.0 {
			t.Errorf("expected 2H Away bet to WIN on native 2H 0-1 score, got %s", resAway.Status)
		}
	})

	// 8. FB_TOTAL_GOALS (2H): refuses to infer or subtract without native 2H period score
	t.Run("FB_TOTAL_GOALS (2H) refuses to infer and produces SETTLEMENT_HOLD if native 2H missing", func(t *testing.T) {
		stateNoNative2H := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-2H-TOTALS",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 3, Away: 1},
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 2, Away: 0},
			},
			ScoreConfirmedAt: now,
		}
		betTotals := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_TOTAL_GOALS",
			Period:              PeriodSecondHalf,
			SelectionCode:       "OVER",
			Line:                floatPtr(1.5),
			RuleVersion:         1,
		}
		res, err := Evaluate(stateNoNative2H, betTotals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold when native 2H score missing for FB_TOTAL_GOALS, got %s", res.Status)
		}
	})

	// 9. All 2H markets are now CERTIFIED and active in capability gate (CanPlaceMarketCanonical returns true)
	t.Run("Second-Half markets are certified and bettable under Full Activation", func(t *testing.T) {
		reqs := []MarketPlacementRequest{
			{SportID: 1, CanonicalMarketCode: "FB_2H_1X2", Period: PeriodSecondHalf},
			{SportID: 1, CanonicalMarketCode: "FB_TOTAL_GOALS", Period: PeriodSecondHalf, Line: floatPtr(1.5)},
		}
		for _, req := range reqs {
			allowed, reason := CanPlaceMarketCanonical(req)
			if !allowed {
				t.Errorf("expected market %s (period %s) to be allowed under Full Activation, got rejected: %s", req.CanonicalMarketCode, req.Period, reason)
			}
		}

		// Rejection applies fail-closed to uncertified/unsupported period (e.g. Q1 on Football)
		badReq := MarketPlacementRequest{SportID: 1, CanonicalMarketCode: "FB_1X2", Period: PeriodQuarter1}
		badAllowed, badReason := CanPlaceMarketCanonical(badReq)
		if badAllowed {
			t.Errorf("expected uncertified period Q1 on Football to fail closed, but was allowed")
		}
		if !strings.Contains(badReason, "suspended") && !strings.Contains(badReason, "Period") {
			t.Errorf("expected rejection reason for unsupported period, got '%s'", badReason)
		}
	})

	// 10. Monotonicity Invariant: Decreasing delta (FT < HT) triggers SettlementHold
	t.Run("Monotonicity Invariant: Decreasing Period Score Delta Triggers SettlementHold", func(t *testing.T) {
		decreasingState := CanonicalEventState{
			CanonicalFixtureID: "SPD-FB-MONO-FAIL",
			SportID:            1,
			Status:             FixtureStatusFT,
			IsFinished:         true,
			Score:              ScoreData{Home: 0, Away: 1}, // Contradiction: Home score decreased from 1 to 0
			PeriodScores: map[string]ScoreData{
				"1H": {Home: 1, Away: 0},
			},
			ScoreConfirmedAt: time.Now(),
		}
		bet2H := BetSelection{
			SportID:             1,
			CanonicalMarketCode: "FB_2H_1X2",
			Period:              PeriodSecondHalf,
			SelectionCode:       "1",
			RuleVersion:         1,
		}
		res, err := Evaluate(decreasingState, bet2H)
		if err != nil {
			t.Fatalf("unexpected evaluate error: %v", err)
		}
		if res.Status != BetStatusSettlementHold {
			t.Errorf("expected BetStatusSettlementHold on decreasing delta, got %s", res.Status)
		}
		if !strings.Contains(res.Reason, "monotonicity violation") && !strings.Contains(res.Reason, "less than half-time") {
			t.Errorf("expected monotonicity violation reason, got '%s'", res.Reason)
		}
	})
}


