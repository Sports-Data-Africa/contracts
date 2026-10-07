package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register American Football (SportID 9) Rules
	Register(&AmericanFootballMoneylineRule{IncludesOT: true})
	Register(&AmericanFootballSpreadRule{IncludesOT: true})
	Register(&AmericanFootballTotalsRule{IncludesOT: true})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter1})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter2})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter3})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter4})
}

// ----------------------------------------------------------------------------
// Moneyline (EVENT_END - includes overtime)
// ----------------------------------------------------------------------------
type AmericanFootballMoneylineRule struct {
	IncludesOT bool
}

func (r *AmericanFootballMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *AmericanFootballMoneylineRule) RequiredDomain() string { return DomainScore }
func (r *AmericanFootballMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *AmericanFootballMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Tied score (moneyline push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Spread (EVENT_END - includes overtime)
// ----------------------------------------------------------------------------
type AmericanFootballSpreadRule struct {
	IncludesOT bool
}

func (r *AmericanFootballSpreadRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_SPREAD", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *AmericanFootballSpreadRule) RequiredDomain() string { return DomainScore }
func (r *AmericanFootballSpreadRule) Timing() string         { return TimingEventEnd }

func (r *AmericanFootballSpreadRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Spread push", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Points (INSTANT_IRREVERSIBLE - includes overtime)
// ----------------------------------------------------------------------------
type AmericanFootballTotalsRule struct {
	IncludesOT bool
}

func (r *AmericanFootballTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_TOTAL_POINTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *AmericanFootballTotalsRule) RequiredDomain() string { return DomainScore }
func (r *AmericanFootballTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *AmericanFootballTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	current := state.Score.Home + state.Score.Away
	line := 45.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("American Football points (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("American Football points (%d) exceeded line (%.1f)", current, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(current) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			} else if float64(current) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(current) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			} else if float64(current) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Quarter Winner (Q1..Q4) (PERIOD_END)
// ----------------------------------------------------------------------------
type AmericanFootballQuarterWinnerRule struct {
	Quarter string
}

func (r *AmericanFootballQuarterWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_QUARTER_WINNER", Period: r.Quarter, RuleVersion: 1}
}
func (r *AmericanFootballQuarterWinnerRule) RequiredDomain() string { return DomainPeriodScore }
func (r *AmericanFootballQuarterWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *AmericanFootballQuarterWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	q, ok := state.PeriodScores[r.Quarter]
	isQuarterDone := state.IsPeriodFinished[r.Quarter] || state.IsFinished

	if !ok || !isQuarterDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}

	h, a := q.Home, q.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "Quarter tied (push)", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}
