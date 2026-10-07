package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Baseball (SportID 11) Rules
	Register(&BaseballMoneylineRule{IncludesExtraInnings: true})
	Register(&BaseballRunLineRule{IncludesExtraInnings: true})
	Register(&BaseballTotalsRule{IncludesExtraInnings: true})
	Register(&BaseballFirst5InningsRule{})
}

// ----------------------------------------------------------------------------
// Baseball Moneyline (EVENT_END - includes extra innings)
// ----------------------------------------------------------------------------
type BaseballMoneylineRule struct {
	IncludesExtraInnings bool
}

func (r *BaseballMoneylineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 11, MarketCode: "BS_MONEYLINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *BaseballMoneylineRule) RequiredDomain() string { return DomainInnings }
func (r *BaseballMoneylineRule) Timing() string         { return TimingEventEnd }

func (r *BaseballMoneylineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Baseball != nil && state.Baseball.IncompleteGameCalled && !state.Baseball.MatchCompleted {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Game called incomplete before minimum innings", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}

	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainInnings}
	}

	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Baseball Run Line (Spread) (EVENT_END - includes extra innings)
// ----------------------------------------------------------------------------
type BaseballRunLineRule struct {
	IncludesExtraInnings bool
}

func (r *BaseballRunLineRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 11, MarketCode: "BS_RUN_LINE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *BaseballRunLineRule) RequiredDomain() string { return DomainInnings }
func (r *BaseballRunLineRule) Timing() string         { return TimingEventEnd }

func (r *BaseballRunLineRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Baseball != nil && state.Baseball.IncompleteGameCalled && !state.Baseball.MatchCompleted {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Run line void on incomplete game", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainInnings}
	}

	spread := -1.5
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
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Run line push", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Game Totals (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type BaseballTotalsRule struct {
	IncludesExtraInnings bool
}

func (r *BaseballTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 11, MarketCode: "BS_TOTAL_RUNS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *BaseballTotalsRule) RequiredDomain() string { return DomainInnings }
func (r *BaseballTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *BaseballTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	currentRuns := state.Score.Home + state.Score.Away
	line := 8.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	// Instant Irreversible: Once runs exceed line, OVER won immediately!
	if sel == "OVER" && float64(currentRuns) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Baseball runs (%d) crossed line (%.1f)", currentRuns, line), RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(currentRuns) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Baseball runs (%d) exceeded line (%.1f)", currentRuns, line), RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}

	// Incomplete game rule for unresolved Under:
	if state.Baseball != nil && state.Baseball.IncompleteGameCalled && !state.Baseball.MatchCompleted {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Incomplete game - under total void", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(currentRuns) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(currentRuns) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainInnings}
}

// ----------------------------------------------------------------------------
// First 5 Innings Winner (PERIOD_END)
// ----------------------------------------------------------------------------
type BaseballFirst5InningsRule struct{}

func (r *BaseballFirst5InningsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 11, MarketCode: "BS_F5_MONEYLINE", Period: "F5", RuleVersion: 1}
}
func (r *BaseballFirst5InningsRule) RequiredDomain() string { return DomainInnings }
func (r *BaseballFirst5InningsRule) Timing() string         { return TimingPeriodEnd }

func (r *BaseballFirst5InningsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Baseball == nil {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainInnings}
	}

	h5, a5 := 0, 0
	f5Completed := 0
	for _, in := range state.Baseball.Innings {
		if in.InningNumber <= 5 && in.IsFinished {
			h5 += in.HomeRuns
			a5 += in.AwayRuns
			f5Completed++
		}
	}

	if f5Completed < 5 {
		if state.Baseball.IncompleteGameCalled {
			return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "Game called before 5 innings completed", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainInnings}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if h5 == a5 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "F5 ended tied (push)", RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}

	won := (sel == "1" || sel == "HOME") && h5 > a5 || (sel == "2" || sel == "AWAY") && a5 > h5
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainInnings, EvaluatedAt: time.Now()}
}
