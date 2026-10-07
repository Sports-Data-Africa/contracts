package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Volleyball (SportID 5) Rules
	Register(&VolleyballMatchWinnerRule{})
	Register(&VolleyballSetWinnerRule{SetNumber: 1, Period: PeriodSet1})
	Register(&VolleyballSetWinnerRule{SetNumber: 2, Period: PeriodSet2})
	Register(&VolleyballSetWinnerRule{SetNumber: 3, Period: PeriodSet3})
	Register(&VolleyballSetWinnerRule{SetNumber: 4, Period: PeriodSet4})
	Register(&VolleyballSetWinnerRule{SetNumber: 5, Period: PeriodSet5})
	Register(&VolleyballTotalPointsRule{})
	Register(&VolleyballSetHandicapRule{})
	Register(&VolleyballPointHandicapRule{})
	Register(&VolleyballCorrectSetScoreRule{})
}

// ----------------------------------------------------------------------------
// Match Winner (Sets won) (EVENT_END)
// ----------------------------------------------------------------------------
type VolleyballMatchWinnerRule struct{}

func (r *VolleyballMatchWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_MATCH_WINNER", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *VolleyballMatchWinnerRule) RequiredDomain() string { return DomainSets }
func (r *VolleyballMatchWinnerRule) Timing() string         { return TimingEventEnd }

func (r *VolleyballMatchWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainSets}
	}
	hSets, aSets := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && hSets > aSets || (sel == "2" || sel == "AWAY") && aSets > hSets
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Set Winner (Set 1..5) (PERIOD_END)
// ----------------------------------------------------------------------------
type VolleyballSetWinnerRule struct {
	SetNumber int
	Period    string
}

func (r *VolleyballSetWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_SET_WINNER", Period: r.Period, RuleVersion: 1}
}
func (r *VolleyballSetWinnerRule) RequiredDomain() string { return DomainPeriodScore }
func (r *VolleyballSetWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *VolleyballSetWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	setKey := fmt.Sprintf("S%d", r.SetNumber)
	sScore, ok := state.PeriodScores[setKey]
	isSetDone := state.IsPeriodFinished[r.Period] || state.IsFinished

	if !ok || !isSetDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}

	h, a := sScore.Home, sScore.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Match Points (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type VolleyballTotalPointsRule struct{}

func (r *VolleyballTotalPointsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_TOTAL_POINTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *VolleyballTotalPointsRule) RequiredDomain() string { return DomainPoints }
func (r *VolleyballTotalPointsRule) Timing() string         { return TimingInstantIrreversible }

func (r *VolleyballTotalPointsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	totalPoints := 0
	for _, p := range state.PeriodScores {
		totalPoints += p.Home + p.Away
	}

	line := 182.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(totalPoints) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Volleyball points (%d) exceeded line (%.1f)", totalPoints, line), RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(totalPoints) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Volleyball points (%d) exceeded line (%.1f)", totalPoints, line), RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(totalPoints) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(totalPoints) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPoints}
}

// ----------------------------------------------------------------------------
// Set Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type VolleyballSetHandicapRule struct{}

func (r *VolleyballSetHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_SET_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *VolleyballSetHandicapRule) RequiredDomain() string { return DomainSets }
func (r *VolleyballSetHandicapRule) Timing() string         { return TimingEventEnd }

func (r *VolleyballSetHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainSets}
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
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Set handicap push", RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Point Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type VolleyballPointHandicapRule struct{}

func (r *VolleyballPointHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_POINT_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *VolleyballPointHandicapRule) RequiredDomain() string { return DomainPoints }
func (r *VolleyballPointHandicapRule) Timing() string         { return TimingEventEnd }

func (r *VolleyballPointHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPoints}
	}
	homePts, awayPts := 0, 0
	for _, p := range state.PeriodScores {
		homePts += p.Home
		awayPts += p.Away
	}

	spread := 0.0
	if bet.Line != nil {
		spread = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	var diff float64
	if sel == "1" || sel == "HOME" {
		diff = (float64(homePts) + spread) - float64(awayPts)
	} else {
		diff = (float64(awayPts) + spread) - float64(homePts)
	}

	if diff > 0 {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Point handicap push", RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPoints, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Correct Set Score (EVENT_END)
// ----------------------------------------------------------------------------
type VolleyballCorrectSetScoreRule struct{}

func (r *VolleyballCorrectSetScoreRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 5, MarketCode: "VB_CORRECT_SET_SCORE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *VolleyballCorrectSetScoreRule) RequiredDomain() string { return DomainSets }
func (r *VolleyballCorrectSetScoreRule) Timing() string         { return TimingEventEnd }

func (r *VolleyballCorrectSetScoreRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainSets}
	}
	target := strings.TrimSpace(bet.SelectionCode)
	actualColon := fmt.Sprintf("%d:%d", state.Score.Home, state.Score.Away)
	actualHyphen := fmt.Sprintf("%d-%d", state.Score.Home, state.Score.Away)

	if target == actualColon || target == actualHyphen {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
}
