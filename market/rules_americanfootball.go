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
	Register(&AmericanFootballTotalsRule{Period: PeriodFullTime, IncludesOT: true})
	Register(&AmericanFootballTotalsRule{Period: PeriodHalfTime, IncludesOT: false})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter1})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter2})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter3})
	Register(&AmericanFootballQuarterWinnerRule{Quarter: PeriodQuarter4})
	Register(&AmericanFootballHalfWinnerRule{})
	Register(&AmericanFootballHalfSpreadRule{})
	Register(&AmericanFootballOddEvenRule{})
	Register(&AmericanFootballTeamTotalsRule{Team: "HOME"})
	Register(&AmericanFootballTeamTotalsRule{Team: "AWAY"})
	Register(&AmericanFootballTeamTotalsRule{Team: ""})
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
// ----------------------------------------------------------------------------
// Total Points (INSTANT_IRREVERSIBLE - includes overtime for FT, regulation for HT)
// ----------------------------------------------------------------------------
type AmericanFootballTotalsRule struct {
	Period     string
	IncludesOT bool
}

func (r *AmericanFootballTotalsRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	return MarketRuleKey{SportID: 9, MarketCode: "AF_TOTAL_POINTS", Period: p, RuleVersion: 1}
}
func (r *AmericanFootballTotalsRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *AmericanFootballTotalsRule) Timing() string { return TimingInstantIrreversible }

func (r *AmericanFootballTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var current int
	var isPeriodDone bool

	if r.Period == PeriodHalfTime {
		ht, ok := state.PeriodScores["1H"]
		if !ok {
			ht, ok = state.PeriodScores["HT"]
		}
		if !ok {
			// Sum Q1 + Q2 if available
			q1, ok1 := state.PeriodScores["Q1"]
			q2, ok2 := state.PeriodScores["Q2"]
			if ok1 && ok2 {
				ht = ScoreData{Home: q1.Home + q2.Home, Away: q1.Away + q2.Away}
				ok = true
			}
		}
		if !ok {
			// Fail-closed: missing 1H period score must NEVER fallback to full-time score
			return SettlementResult{
				Status:         BetStatusSettlementHold,
				Settled:        false,
				Reason:         "Missing 1H period score data for American Football half-time totals",
				Timing:         TimingPeriodEnd,
				RequiredDomain: DomainPeriodScore,
			}
		}
		current = ht.Home + ht.Away
		isPeriodDone = state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	} else {
		current = state.Score.Home + state.Score.Away
		isPeriodDone = state.IsFinished
	}

	line := 45.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("American Football points (%d) exceeded line (%.1f)", current, line), RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("American Football points (%d) exceeded line (%.1f)", current, line), RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
	}

	if isPeriodDone {
		switch sel {
		case "OVER":
			if float64(current) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(current) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(current) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(current) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: r.RequiredDomain()}
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

// ----------------------------------------------------------------------------
// 1st Half Winner (PERIOD_END)
// ----------------------------------------------------------------------------
type AmericanFootballHalfWinnerRule struct{}

func (r *AmericanFootballHalfWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_1H_WINNER", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *AmericanFootballHalfWinnerRule) RequiredDomain() string { return DomainPeriodScore }
func (r *AmericanFootballHalfWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *AmericanFootballHalfWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores[PeriodHalfTime]
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.IsFinished
	if !ok {
		q1, ok1 := state.PeriodScores[PeriodQuarter1]
		q2, ok2 := state.PeriodScores[PeriodQuarter2]
		if ok1 && ok2 && (state.IsPeriodFinished[PeriodQuarter2] || state.IsFinished) {
			ht = ScoreData{Home: q1.Home + q2.Home, Away: q1.Away + q2.Away}
			isHTDone = true
			ok = true
		}
	}
	if !ok || !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	h, a := ht.Home, ht.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "1st Half tied (push)", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Spread (PERIOD_END)
// ----------------------------------------------------------------------------
type AmericanFootballHalfSpreadRule struct{}

func (r *AmericanFootballHalfSpreadRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_1H_SPREAD", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *AmericanFootballHalfSpreadRule) RequiredDomain() string { return DomainPeriodScore }
func (r *AmericanFootballHalfSpreadRule) Timing() string         { return TimingPeriodEnd }

func (r *AmericanFootballHalfSpreadRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores[PeriodHalfTime]
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.IsFinished
	if !ok {
		q1, ok1 := state.PeriodScores[PeriodQuarter1]
		q2, ok2 := state.PeriodScores[PeriodQuarter2]
		if ok1 && ok2 && (state.IsPeriodFinished[PeriodQuarter2] || state.IsFinished) {
			ht = ScoreData{Home: q1.Home + q2.Home, Away: q1.Away + q2.Away}
			isHTDone = true
			ok = true
		}
	}
	if !ok || !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	spread := 0.0
	if bet.Line != nil {
		spread = *bet.Line
	}
	h, a := float64(ht.Home), float64(ht.Away)
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	var diff float64
	if sel == "1" || sel == "HOME" {
		diff = (h + spread) - a
	} else {
		diff = (a + spread) - h
	}
	if diff > 0 {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "1H spread push", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Odd/Even Total Points (EVENT_END)
// ----------------------------------------------------------------------------
type AmericanFootballOddEvenRule struct{}

func (r *AmericanFootballOddEvenRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 9, MarketCode: "AF_ODD_EVEN", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *AmericanFootballOddEvenRule) RequiredDomain() string { return DomainScore }
func (r *AmericanFootballOddEvenRule) Timing() string         { return TimingEventEnd }

func (r *AmericanFootballOddEvenRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	tot := state.Score.Home + state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isEven := tot%2 == 0
	won := (isEven && (sel == "EVEN" || sel == "E")) || (!isEven && (sel == "ODD" || sel == "O"))
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Team Totals (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type AmericanFootballTeamTotalsRule struct {
	Team string // "HOME" or "AWAY"
}

func (r *AmericanFootballTeamTotalsRule) Key() MarketRuleKey {
	code := "AF_TEAM_TOTALS"
	switch strings.ToUpper(r.Team) {
	case "HOME":
		code = "AF_HOME_TOTALS"
	case "AWAY":
		code = "AF_AWAY_TOTALS"
	}
	return MarketRuleKey{SportID: 9, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *AmericanFootballTeamTotalsRule) RequiredDomain() string { return DomainScore }
func (r *AmericanFootballTeamTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *AmericanFootballTeamTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	points := state.Score.Home
	if strings.ToUpper(r.Team) == "AWAY" {
		points = state.Score.Away
	}
	line := 21.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel == "OVER" && float64(points) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(points) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if state.IsFinished {
		if sel == "OVER" {
			if float64(points) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			} else if float64(points) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if sel == "UNDER" {
			if float64(points) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			} else if float64(points) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}
	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}
