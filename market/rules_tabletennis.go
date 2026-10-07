package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Table Tennis (SportID 13) Rules
	Register(&TableTennisMatchWinnerRule{})
	Register(&TableTennisGameWinnerRule{GameNumber: 1, Period: "GAME1"})
	Register(&TableTennisGameWinnerRule{GameNumber: 2, Period: "GAME2"})
	Register(&TableTennisGameWinnerRule{GameNumber: 3, Period: "GAME3"})
	Register(&TableTennisGameWinnerRule{GameNumber: 4, Period: "GAME4"})
	Register(&TableTennisGameWinnerRule{GameNumber: 5, Period: "GAME5"})
	Register(&TableTennisTotalPointsRule{})
	Register(&TableTennisPointHandicapRule{})
	Register(&TableTennisGameHandicapRule{})
	Register(&TableTennisCorrectScoreRule{})
}

// ----------------------------------------------------------------------------
// Table Tennis Match Winner (EVENT_END)
// ----------------------------------------------------------------------------
type TableTennisMatchWinnerRule struct{}

func (r *TableTennisMatchWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_MATCH_WINNER", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TableTennisMatchWinnerRule) RequiredDomain() string { return DomainGames }
func (r *TableTennisMatchWinnerRule) Timing() string         { return TimingEventEnd }

func (r *TableTennisMatchWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainGames}
	}
	hGames, aGames := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && hGames > aGames || (sel == "2" || sel == "AWAY") && aGames > hGames
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Game Winner (Game 1..5) (PERIOD_END)
// ----------------------------------------------------------------------------
type TableTennisGameWinnerRule struct {
	GameNumber int
	Period     string
}

func (r *TableTennisGameWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_GAME_WINNER", Period: r.Period, RuleVersion: 1}
}
func (r *TableTennisGameWinnerRule) RequiredDomain() string { return DomainPoints }
func (r *TableTennisGameWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *TableTennisGameWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	gKey := fmt.Sprintf("G%d", r.GameNumber)
	gScore, ok := state.PeriodScores[gKey]
	isGameDone := state.IsPeriodFinished[r.Period] || state.IsFinished

	if !ok || !isGameDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPoints}
	}

	h, a := gScore.Home, gScore.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Match Points (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type TableTennisTotalPointsRule struct{}

func (r *TableTennisTotalPointsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_TOTAL_POINTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TableTennisTotalPointsRule) RequiredDomain() string { return DomainPoints }
func (r *TableTennisTotalPointsRule) Timing() string         { return TimingInstantIrreversible }

func (r *TableTennisTotalPointsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	totalPts := 0
	for _, g := range state.PeriodScores {
		totalPts += g.Home + g.Away
	}

	line := 74.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(totalPts) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Table Tennis points (%d) exceeded line (%.1f)", totalPts, line), RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(totalPts) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Table Tennis points (%d) exceeded line (%.1f)", totalPts, line), RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(totalPts) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(totalPts) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPoints}
}

// ----------------------------------------------------------------------------
// Point Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type TableTennisPointHandicapRule struct{}

func (r *TableTennisPointHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_POINT_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TableTennisPointHandicapRule) RequiredDomain() string { return DomainPoints }
func (r *TableTennisPointHandicapRule) Timing() string         { return TimingEventEnd }

func (r *TableTennisPointHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPoints}
	}
	hPts, aPts := 0, 0
	for _, g := range state.PeriodScores {
		hPts += g.Home
		aPts += g.Away
	}

	spread := 0.0
	if bet.Line != nil {
		spread = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	var diff float64
	if sel == "1" || sel == "HOME" {
		diff = (float64(hPts) + spread) - float64(aPts)
	} else {
		diff = (float64(aPts) + spread) - float64(hPts)
	}

	if diff > 0 {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Point handicap push", RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Game Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type TableTennisGameHandicapRule struct{}

func (r *TableTennisGameHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_GAME_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TableTennisGameHandicapRule) RequiredDomain() string { return DomainGames }
func (r *TableTennisGameHandicapRule) Timing() string         { return TimingEventEnd }

func (r *TableTennisGameHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainGames}
	}
	spread := 0.0
	if bet.Line != nil {
		spread = *bet.Line
	}
	h, a := float64(state.Score.Home), float64(state.Score.Away)
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	var diff float64
	if sel == "1" || sel == "HOME" {
		diff = (h + spread) - a
	} else {
		diff = (a + spread) - h
	}

	if diff > 0 {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Game handicap push", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Correct Score (EVENT_END)
// ----------------------------------------------------------------------------
type TableTennisCorrectScoreRule struct{}

func (r *TableTennisCorrectScoreRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 13, MarketCode: "TT_CORRECT_SCORE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TableTennisCorrectScoreRule) RequiredDomain() string { return DomainGames }
func (r *TableTennisCorrectScoreRule) Timing() string         { return TimingEventEnd }

func (r *TableTennisCorrectScoreRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainGames}
	}
	target := strings.TrimSpace(bet.SelectionCode)
	actualColon := fmt.Sprintf("%d:%d", state.Score.Home, state.Score.Away)
	actualHyphen := fmt.Sprintf("%d-%d", state.Score.Home, state.Score.Away)

	if target == actualColon || target == actualHyphen {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
}
