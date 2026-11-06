package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Ice Hockey (SportID 4) Rules
	Register(&IceHockeyRegulation1X2Rule{})
	Register(&IceHockeyMoneylineRule{IncludesOT: true, IncludesSO: true})
	Register(&IceHockeyPuckLineRule{IncludesOT: true})
	Register(&IceHockeyTotalsRule{Period: PeriodFullTime, IncludesOT: true})
	Register(&IceHockeyPeriodTotalsRule{Period: PeriodPeriod1})
	Register(&IceHockeyPeriodTotalsRule{Period: PeriodPeriod2})
	Register(&IceHockeyPeriodTotalsRule{Period: PeriodPeriod3})
	Register(&IceHockeyPeriod1X2Rule{Period: PeriodPeriod1})
	Register(&IceHockeyPeriod1X2Rule{Period: PeriodPeriod2})
	Register(&IceHockeyPeriod1X2Rule{Period: PeriodPeriod3})
	Register(&IceHockeyOddEvenRule{})
	Register(&IceHockeyTeamTotalsRule{Team: "HOME"})
	Register(&IceHockeyTeamTotalsRule{Team: "AWAY"})
	Register(&IceHockeyTeamTotalsRule{Team: ""})
	Register(&IceHockeyHighestScoringPeriodRule{})
}

// ----------------------------------------------------------------------------
// Regulation 1X2 (EVENT_END - EXCLUDES OT and Shootout)
// ----------------------------------------------------------------------------
type IceHockeyRegulation1X2Rule struct{}

func (r *IceHockeyRegulation1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_REGULATION_1X2", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyRegulation1X2Rule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyRegulation1X2Rule) Timing() string         { return TimingEventEnd }

func (r *IceHockeyRegulation1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	// Regulation score: P1 + P2 + P3
	p1 := state.PeriodScores["P1"]
	p2 := state.PeriodScores["P2"]
	p3 := state.PeriodScores["P3"]

	h := p1.Home + p2.Home + p3.Home
	a := p1.Away + p2.Away + p3.Away
	if h == 0 && a == 0 && (state.Score.Home > 0 || state.Score.Away > 0) && state.ExtraTime == nil {
		h = state.Score.Home
		a = state.Score.Away
	}

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
// Moneyline (EVENT_END - INCLUDES Overtime and Shootout)
// ----------------------------------------------------------------------------
type IceHockeyMoneylineRule struct {
	IncludesOT bool
	IncludesSO bool
}

func (r *IceHockeyMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyMoneylineRule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *IceHockeyMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Puck Line (EVENT_END - includes overtime)
// ----------------------------------------------------------------------------
type IceHockeyPuckLineRule struct {
	IncludesOT bool
}

func (r *IceHockeyPuckLineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_PUCK_LINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyPuckLineRule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyPuckLineRule) Timing() string         { return TimingEventEnd }

func (r *IceHockeyPuckLineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Puck line push", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Totals Over/Under (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type IceHockeyTotalsRule struct {
	Period     string
	IncludesOT bool
}

func (r *IceHockeyTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_TOTAL_GOALS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyTotalsRule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *IceHockeyTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	currentGoals := state.Score.Home + state.Score.Away
	line := 5.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(currentGoals) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Hockey total goals (%d) crossed line (%.1f)", currentGoals, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(currentGoals) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Hockey total goals (%d) exceeded line (%.1f)", currentGoals, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(currentGoals) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(currentGoals) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Period 1X2 (PERIOD_END)
// ----------------------------------------------------------------------------
type IceHockeyPeriod1X2Rule struct {
	Period string // PERIOD1, PERIOD2, PERIOD3
}

func (r *IceHockeyPeriod1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_PERIOD_1X2", Period: r.Period, RuleVersion: 1}
}
func (r *IceHockeyPeriod1X2Rule) RequiredDomain() string { return DomainPeriodScore }
func (r *IceHockeyPeriod1X2Rule) Timing() string         { return TimingPeriodEnd }

func (r *IceHockeyPeriod1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	pKey := "P1"
	switch r.Period {
	case PeriodPeriod2:
		pKey = "P2"
	case PeriodPeriod3:
		pKey = "P3"
	}
	pScore, ok := state.PeriodScores[pKey]
	isPeriodDone := state.IsPeriodFinished[r.Period] || state.IsFinished

	if !ok || !isPeriodDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}

	h, a := pScore.Home, pScore.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Period Totals (INSTANT_IRREVERSIBLE / PERIOD_END)
// ----------------------------------------------------------------------------
type IceHockeyPeriodTotalsRule struct {
	Period string
}

func (r *IceHockeyPeriodTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_PERIOD_TOTALS", Period: r.Period, RuleVersion: 1}
}
func (r *IceHockeyPeriodTotalsRule) RequiredDomain() string { return DomainPeriodScore }
func (r *IceHockeyPeriodTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *IceHockeyPeriodTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	pKey := "P1"
	switch r.Period {
	case PeriodPeriod2:
		pKey = "P2"
	case PeriodPeriod3:
		pKey = "P3"
	}
	pScore, ok := state.PeriodScores[pKey]
	isPeriodDone := state.IsPeriodFinished[r.Period] || state.IsFinished

	current := 0
	if ok {
		current = pScore.Home + pScore.Away
	}

	line := 1.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(current) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("%s goals (%d) crossed line (%.1f)", r.Period, current, line), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(current) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("%s goals (%d) exceeded line (%.1f)", r.Period, current, line), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	if isPeriodDone {
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
// Odd/Even Total Goals (EVENT_END)
// ----------------------------------------------------------------------------
type IceHockeyOddEvenRule struct{}

func (r *IceHockeyOddEvenRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_ODD_EVEN", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyOddEvenRule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyOddEvenRule) Timing() string         { return TimingEventEnd }

func (r *IceHockeyOddEvenRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
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
type IceHockeyTeamTotalsRule struct {
	Team string // "HOME" or "AWAY"
}

func (r *IceHockeyTeamTotalsRule) Key() MarketRuleKey {
	code := "IH_TEAM_TOTALS"
	switch strings.ToUpper(r.Team) {
	case "HOME":
		code = "IH_HOME_TOTALS"
	case "AWAY":
		code = "IH_AWAY_TOTALS"
	}
	return MarketRuleKey{SportID: 4, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyTeamTotalsRule) RequiredDomain() string { return DomainScore }
func (r *IceHockeyTeamTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *IceHockeyTeamTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	goals := state.Score.Home
	if strings.ToUpper(r.Team) == "AWAY" {
		goals = state.Score.Away
	}
	line := 2.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel == "OVER" && float64(goals) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(goals) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if state.IsFinished {
		if sel == "OVER" {
			if float64(goals) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if sel == "UNDER" {
			if float64(goals) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}
	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Highest Scoring Period (EVENT_END - P1, P2, P3, EQUAL)
// ----------------------------------------------------------------------------
type IceHockeyHighestScoringPeriodRule struct{}

func (r *IceHockeyHighestScoringPeriodRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 4, MarketCode: "IH_HIGHEST_SCORING_PERIOD", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *IceHockeyHighestScoringPeriodRule) RequiredDomain() string { return DomainPeriodScore }
func (r *IceHockeyHighestScoringPeriodRule) Timing() string         { return TimingEventEnd }

func (r *IceHockeyHighestScoringPeriodRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	p1Score := state.PeriodScores["P1"]
	p2Score := state.PeriodScores["P2"]
	p3Score := state.PeriodScores["P3"]

	p1Total := p1Score.Home + p1Score.Away
	p2Total := p2Score.Home + p2Score.Away
	p3Total := p3Score.Home + p3Score.Away

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	var winningSel string
	if p1Total > p2Total && p1Total > p3Total {
		winningSel = "1"
	} else if p2Total > p1Total && p2Total > p3Total {
		winningSel = "2"
	} else if p3Total > p1Total && p3Total > p2Total {
		winningSel = "3"
	} else {
		winningSel = "EQUAL"
	}

	won := (sel == winningSel) ||
		(winningSel == "1" && (sel == "P1" || sel == "PERIOD 1")) ||
		(winningSel == "2" && (sel == "P2" || sel == "PERIOD 2")) ||
		(winningSel == "3" && (sel == "P3" || sel == "PERIOD 3")) ||
		(winningSel == "EQUAL" && (sel == "DRAW" || sel == "TIE" || sel == "X"))

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}
