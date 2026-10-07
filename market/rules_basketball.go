package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Basketball (SportID 2) Rules
	Register(&BasketballMoneylineRule{IncludesOT: true})
	Register(&BasketballSpreadRule{IncludesOT: true})
	Register(&BasketballTotalsRule{Period: PeriodFullTime, IncludesOT: true})
	Register(&BasketballTotalsRule{Period: PeriodHalfTime, IncludesOT: false})
	Register(&BasketballQuarterTotalsRule{Quarter: PeriodQuarter1})
	Register(&BasketballQuarterTotalsRule{Quarter: PeriodQuarter2})
	Register(&BasketballQuarterTotalsRule{Quarter: PeriodQuarter3})
	Register(&BasketballQuarterTotalsRule{Quarter: PeriodQuarter4})
	Register(&BasketballQuarterWinnerRule{Quarter: PeriodQuarter1})
	Register(&BasketballQuarterWinnerRule{Quarter: PeriodQuarter2})
	Register(&BasketballQuarterWinnerRule{Quarter: PeriodQuarter3})
	Register(&BasketballQuarterWinnerRule{Quarter: PeriodQuarter4})
	Register(&BasketballHalfWinnerRule{})
}

// ----------------------------------------------------------------------------
// Basketball Moneyline (EVENT_END - includes overtime)
// ----------------------------------------------------------------------------
type BasketballMoneylineRule struct {
	IncludesOT bool
}

func (r *BasketballMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 2, MarketCode: "BB_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *BasketballMoneylineRule) RequiredDomain() string { return DomainScore }
func (r *BasketballMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *BasketballMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
// Basketball Spread (EVENT_END - includes overtime)
// ----------------------------------------------------------------------------
type BasketballSpreadRule struct {
	IncludesOT bool
}

func (r *BasketballSpreadRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 2, MarketCode: "BB_SPREAD", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *BasketballSpreadRule) RequiredDomain() string { return DomainScore }
func (r *BasketballSpreadRule) Timing() string         { return TimingEventEnd }

func (r *BasketballSpreadRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Spread landed exactly on handicap line (push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Basketball Total Points (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type BasketballTotalsRule struct {
	Period     string
	IncludesOT bool
}

func (r *BasketballTotalsRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	return MarketRuleKey{SportID: 2, MarketCode: "BB_TOTAL_POINTS", Period: p, RuleVersion: 1}
}
func (r *BasketballTotalsRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *BasketballTotalsRule) Timing() string { return TimingInstantIrreversible }

func (r *BasketballTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var currentPoints int
	var periodDone bool

	if r.Period == PeriodHalfTime {
		if ht, ok := state.PeriodScores["1H"]; ok {
			currentPoints = ht.Home + ht.Away
		} else if ht, ok := state.PeriodScores["HT"]; ok {
			currentPoints = ht.Home + ht.Away
		} else {
			// Sum Q1 + Q2
			q1 := state.PeriodScores["Q1"]
			q2 := state.PeriodScores["Q2"]
			currentPoints = q1.Home + q1.Away + q2.Home + q2.Away
		}
		periodDone = state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	} else {
		currentPoints = state.Score.Home + state.Score.Away
		periodDone = state.IsFinished
	}

	line := 215.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	// Instant Irreversible threshold checks
	if sel == "OVER" && float64(currentPoints) > line {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Basketball total points (%d) exceeded line (%.1f)", currentPoints, line),
			RequiredDomain: r.RequiredDomain(),
			EvaluatedAt:    time.Now(),
		}
	}
	if sel == "UNDER" && float64(currentPoints) > line {
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Basketball total points (%d) exceeded under line (%.1f)", currentPoints, line),
			RequiredDomain: r.RequiredDomain(),
			EvaluatedAt:    time.Now(),
		}
	}

	if periodDone {
		switch sel {
		case "OVER":
			if float64(currentPoints) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(currentPoints) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(currentPoints) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(currentPoints) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: r.RequiredDomain()}
}

// ----------------------------------------------------------------------------
// Quarter Totals (Q1-Q4) (INSTANT_IRREVERSIBLE / PERIOD_END)
// ----------------------------------------------------------------------------
type BasketballQuarterTotalsRule struct {
	Quarter string
}

func (r *BasketballQuarterTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 2, MarketCode: "BB_QUARTER_TOTALS", Period: r.Quarter, RuleVersion: 1}
}
func (r *BasketballQuarterTotalsRule) RequiredDomain() string { return DomainPeriodScore }
func (r *BasketballQuarterTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *BasketballQuarterTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	qScore, ok := state.PeriodScores[r.Quarter]
	isQuarterDone := state.IsPeriodFinished[r.Quarter] || state.IsFinished

	current := 0
	if ok {
		current = qScore.Home + qScore.Away
	}

	line := 52.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("%s points (%d) crossed line (%.1f)", r.Quarter, current, line), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("%s points (%d) exceeded line (%.1f)", r.Quarter, current, line), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	if isQuarterDone {
		switch sel {
		case "OVER":
			if float64(current) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(current) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPeriodScore}
}

// ----------------------------------------------------------------------------
// Quarter Winner (Q1-Q4) (PERIOD_END)
// ----------------------------------------------------------------------------
type BasketballQuarterWinnerRule struct {
	Quarter string
}

func (r *BasketballQuarterWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 2, MarketCode: "BB_QUARTER_WINNER", Period: r.Quarter, RuleVersion: 1}
}
func (r *BasketballQuarterWinnerRule) RequiredDomain() string { return DomainPeriodScore }
func (r *BasketballQuarterWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *BasketballQuarterWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	qScore, ok := state.PeriodScores[r.Quarter]
	isQuarterDone := state.IsPeriodFinished[r.Quarter] || state.IsFinished

	if !ok || !isQuarterDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}

	h, a := qScore.Home, qScore.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: fmt.Sprintf("%s tied (push)", r.Quarter), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// First Half Winner (PERIOD_END)
// ----------------------------------------------------------------------------
type BasketballHalfWinnerRule struct{}

func (r *BasketballHalfWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 2, MarketCode: "BB_1H_WINNER", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *BasketballHalfWinnerRule) RequiredDomain() string { return DomainPeriodScore }
func (r *BasketballHalfWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *BasketballHalfWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "1H ended tied (push)", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}
