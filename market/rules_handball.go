package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Handball (SportID 10) Rules
	Register(&Handball1X2Rule{})
	Register(&HandballMoneylineRule{})
	Register(&HandballHandicapRule{})
	Register(&HandballTotalsRule{})
	Register(&HandballHT1X2Rule{})
}

// ----------------------------------------------------------------------------
// Handball 1X2 (EVENT_END - Regulation 60 mins)
// ----------------------------------------------------------------------------
type Handball1X2Rule struct{}

func (r *Handball1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 10, MarketCode: "HB_1X2", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *Handball1X2Rule) RequiredDomain() string { return DomainScore }
func (r *Handball1X2Rule) Timing() string         { return TimingEventEnd }

func (r *Handball1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
// Handball Moneyline (EVENT_END)
// ----------------------------------------------------------------------------
type HandballMoneylineRule struct{}

func (r *HandballMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 10, MarketCode: "HB_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *HandballMoneylineRule) RequiredDomain() string { return DomainScore }
func (r *HandballMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *HandballMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Handball moneyline push on draw", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Handball Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type HandballHandicapRule struct{}

func (r *HandballHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 10, MarketCode: "HB_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *HandballHandicapRule) RequiredDomain() string { return DomainScore }
func (r *HandballHandicapRule) Timing() string         { return TimingEventEnd }

func (r *HandballHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Handball handicap push", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Handball Total Goals (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type HandballTotalsRule struct{}

func (r *HandballTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 10, MarketCode: "HB_TOTAL_GOALS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *HandballTotalsRule) RequiredDomain() string { return DomainScore }
func (r *HandballTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *HandballTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	current := state.Score.Home + state.Score.Away
	line := 55.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Handball goals (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Handball goals (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
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
// Handball Half Time 1X2 (PERIOD_END)
// ----------------------------------------------------------------------------
type HandballHT1X2Rule struct{}

func (r *HandballHT1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 10, MarketCode: "HB_HT_1X2", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *HandballHT1X2Rule) RequiredDomain() string { return DomainPeriodScore }
func (r *HandballHT1X2Rule) Timing() string         { return TimingPeriodEnd }

func (r *HandballHT1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
