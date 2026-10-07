package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Rugby (SportID 7) Rules
	Register(&Rugby1X2Rule{})
	Register(&RugbyMoneylineRule{IncludesET: true})
	Register(&RugbyHandicapRule{})
	Register(&RugbyTotalsRule{})
	Register(&RugbyHT1X2Rule{})
}

// ----------------------------------------------------------------------------
// Rugby 1X2 / 3-Way (EVENT_END - Regulation 80 mins)
// ----------------------------------------------------------------------------
type Rugby1X2Rule struct{}

func (r *Rugby1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 7, MarketCode: "RB_1X2", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *Rugby1X2Rule) RequiredDomain() string { return DomainScore }
func (r *Rugby1X2Rule) Timing() string         { return TimingEventEnd }

func (r *Rugby1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Rugby Moneyline (EVENT_END - includes Extra Time if played)
// ----------------------------------------------------------------------------
type RugbyMoneylineRule struct {
	IncludesET bool
}

func (r *RugbyMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 7, MarketCode: "RB_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *RugbyMoneylineRule) RequiredDomain() string { return DomainScore }
func (r *RugbyMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *RugbyMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Rugby moneyline push on draw", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Rugby Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type RugbyHandicapRule struct{}

func (r *RugbyHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 7, MarketCode: "RB_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *RugbyHandicapRule) RequiredDomain() string { return DomainScore }
func (r *RugbyHandicapRule) Timing() string         { return TimingEventEnd }

func (r *RugbyHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
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
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Handicap push", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Rugby Total Points (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type RugbyTotalsRule struct{}

func (r *RugbyTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 7, MarketCode: "RB_TOTAL_POINTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *RugbyTotalsRule) RequiredDomain() string { return DomainScore }
func (r *RugbyTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *RugbyTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	current := state.Score.Home + state.Score.Away
	line := 41.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Rugby points (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Rugby points (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(current) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(current) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Rugby Half Time 1X2 (PERIOD_END)
// ----------------------------------------------------------------------------
type RugbyHT1X2Rule struct{}

func (r *RugbyHT1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 7, MarketCode: "RB_HT_1X2", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *RugbyHT1X2Rule) RequiredDomain() string { return DomainPeriodScore }
func (r *RugbyHT1X2Rule) Timing() string         { return TimingPeriodEnd }

func (r *RugbyHT1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished

	if !ok || !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}

	h, a := ht.Home, ht.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}
