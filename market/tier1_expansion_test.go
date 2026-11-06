package market_test

import (
	"testing"

	"github.com/Sports-Data-Africa/contracts/market"
)

func TestTier1_ExpansionRules(t *testing.T) {
	// 1. Basketball Quarter Spread (BB_QUARTER_SPREAD:Q1)
	t.Run("Basketball_Q1_Spread", func(t *testing.T) {
		rule, ok := market.Get(2, "BB_QUARTER_SPREAD", market.PeriodQuarter1, 1)
		if !ok {
			t.Fatalf("Failed to resolve BB_QUARTER_SPREAD rule")
		}
		spreadLine := -2.5
		bet := market.BetSelection{
			CanonicalMarketCode: "BB_QUARTER_SPREAD",
			Period:              market.PeriodQuarter1,
			SelectionCode:       "1",
			Line:                &spreadLine,
		}
		state := market.CanonicalEventState{
			PeriodScores: map[string]market.ScoreData{
				market.PeriodQuarter1: {Home: 28, Away: 24},
			},
			IsPeriodFinished: map[string]bool{
				market.PeriodQuarter1: true,
			},
		}
		res := rule.Evaluate(state, bet)
		if res.Status != market.BetStatusWon {
			t.Errorf("Expected Won (28 - 2.5 = 25.5 > 24), got %s", res.Status)
		}
	})

	// 2. Handball Winning Margin Bands (HB_WINNING_MARGIN:FT)
	t.Run("Handball_Winning_Margin", func(t *testing.T) {
		rule, ok := market.Get(10, "HB_WINNING_MARGIN", market.PeriodFullTime, 1)
		if !ok {
			t.Fatalf("Failed to resolve HB_WINNING_MARGIN rule")
		}
		bet := market.BetSelection{
			CanonicalMarketCode: "HB_WINNING_MARGIN",
			Period:              market.PeriodFullTime,
			SelectionCode:       "HOME_3_5",
		}
		state := market.CanonicalEventState{
			IsFinished: true,
			Score: market.ScoreData{
				Home: 31,
				Away: 27, // diff = 4 -> HOME_3_5
			},
		}
		res := rule.Evaluate(state, bet)
		if res.Status != market.BetStatusWon {
			t.Errorf("Expected Won for 4 goal margin in band HOME_3_5, got %s", res.Status)
		}
	})

	// 3. Ice Hockey Highest Scoring Period (IH_HIGHEST_SCORING_PERIOD:FT)
	t.Run("Ice_Hockey_Highest_Scoring_Period", func(t *testing.T) {
		rule, ok := market.Get(4, "IH_HIGHEST_SCORING_PERIOD", market.PeriodFullTime, 1)
		if !ok {
			t.Fatalf("Failed to resolve IH_HIGHEST_SCORING_PERIOD rule")
		}
		bet := market.BetSelection{
			CanonicalMarketCode: "IH_HIGHEST_SCORING_PERIOD",
			Period:              market.PeriodFullTime,
			SelectionCode:       "2", // Period 2
		}
		state := market.CanonicalEventState{
			IsFinished: true,
			PeriodScores: map[string]market.ScoreData{
				"P1": {Home: 1, Away: 0}, // 1 goal
				"P2": {Home: 2, Away: 2}, // 4 goals (highest)
				"P3": {Home: 1, Away: 1}, // 2 goals
			},
		}
		res := rule.Evaluate(state, bet)
		if res.Status != market.BetStatusWon {
			t.Errorf("Expected Won for P2 highest scoring period, got %s", res.Status)
		}
	})

	// 4. Baseball First 5 Innings Run Line (BS_F5_RUN_LINE:F5)
	t.Run("Baseball_F5_Run_Line", func(t *testing.T) {
		rule, ok := market.Get(11, "BS_F5_RUN_LINE", "F5", 1)
		if !ok {
			t.Fatalf("Failed to resolve BS_F5_RUN_LINE rule")
		}
		runLine := -0.5
		bet := market.BetSelection{
			CanonicalMarketCode: "BS_F5_RUN_LINE",
			Period:              "F5",
			SelectionCode:       "1",
			Line:                &runLine,
		}
		state := market.CanonicalEventState{
			Baseball: &market.BaseballStateData{
				CurrentInning: 6,
				Innings: []market.BaseballInningData{
					{InningNumber: 1, HomeRuns: 1, AwayRuns: 0, IsFinished: true},
					{InningNumber: 2, HomeRuns: 0, AwayRuns: 0, IsFinished: true},
					{InningNumber: 3, HomeRuns: 2, AwayRuns: 1, IsFinished: true},
					{InningNumber: 4, HomeRuns: 0, AwayRuns: 0, IsFinished: true},
					{InningNumber: 5, HomeRuns: 1, AwayRuns: 1, IsFinished: true},
				},
			},
			PeriodScores: map[string]market.ScoreData{
				"F5": {Home: 4, Away: 2},
			},
			IsPeriodFinished: map[string]bool{
				"F5": true,
			},
		}
		res := rule.Evaluate(state, bet)
		if res.Status != market.BetStatusWon {
			t.Errorf("Expected Won for F5 run line Home -0.5 (4 - 2), got %s", res.Status)
		}
	})
}
