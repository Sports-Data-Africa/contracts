package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Cricket (SportID 8) Rules
	Register(&CricketMatchWinnerRule{})
	Register(&CricketTeamRunsRule{})
	Register(&CricketMostSixesRule{})
	Register(&CricketMostFoursRule{})
}

// ----------------------------------------------------------------------------
// Match Winner (EVENT_END with DLS / Rain handling)
// ----------------------------------------------------------------------------
type CricketMatchWinnerRule struct{}

func (r *CricketMatchWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 8, MarketCode: "CK_MATCH_WINNER", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *CricketMatchWinnerRule) RequiredDomain() string { return DomainScore }
func (r *CricketMatchWinnerRule) Timing() string         { return TimingEventEnd }

func (r *CricketMatchWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Cricket == nil {
		if !state.IsFinished {
			return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
		}
		// Fallback score
		h, a := state.Score.Home, state.Score.Away
		sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
		if (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	if !state.Cricket.MatchCompleted && !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}

	winner := state.Cricket.Winner
	if winner == "NO_RESULT" || winner == "ABANDONED" {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Cricket match abandoned / No Result", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if winner == "TIE" || winner == "DRAW" {
		// Tie rule: if super over played, super over determines winner
		if state.Cricket.SuperOverScore != nil {
			soH, soA := state.Cricket.SuperOverScore.Home, state.Cricket.SuperOverScore.Away
			if soH > soA {
				winner = "1"
			} else if soA > soH {
				winner = "2"
			}
		}
		if winner == "TIE" || winner == "DRAW" {
			return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Match tied/drawn without super over winner", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1" || sel == "HOME") && winner == "1" || (sel == "2" || sel == "AWAY") && winner == "2"

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Team Runs / Innings Totals (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type CricketTeamRunsRule struct{}

func (r *CricketTeamRunsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 8, MarketCode: "CK_TEAM_RUNS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *CricketTeamRunsRule) RequiredDomain() string { return DomainScore }
func (r *CricketTeamRunsRule) Timing() string         { return TimingInstantIrreversible }

func (r *CricketTeamRunsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	runs := 0
	if state.Cricket != nil && len(state.Cricket.Innings) > 0 {
		// Sum runs for the relevant innings
		for _, inn := range state.Cricket.Innings {
			runs += inn.Runs
		}
	} else {
		runs = state.Score.Home + state.Score.Away
	}

	line := 160.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(runs) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Cricket runs (%d) crossed line (%.1f)", runs, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(runs) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Cricket runs (%d) exceeded line (%.1f)", runs, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	// Rain reduction / DLS check for unresolved unders:
	if state.Cricket != nil && state.Cricket.DLSApplied && !state.Cricket.MatchCompleted {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
	}

	if state.IsFinished || (state.Cricket != nil && state.Cricket.MatchCompleted) {
		switch sel {
		case "OVER":
			if float64(runs) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(runs) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Most Sixes (EVENT_END)
// ----------------------------------------------------------------------------
type CricketMostSixesRule struct{}

func (r *CricketMostSixesRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 8, MarketCode: "CK_MOST_SIXES", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *CricketMostSixesRule) RequiredDomain() string { return DomainTeamStats }
func (r *CricketMostSixesRule) Timing() string         { return TimingEventEnd }

func (r *CricketMostSixesRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Cricket == nil || (!state.Cricket.MatchCompleted && !state.IsFinished) {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats}
	}
	h, a := state.Cricket.Sixes.Home, state.Cricket.Sixes.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "TIE" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Most Fours (EVENT_END)
// ----------------------------------------------------------------------------
type CricketMostFoursRule struct{}

func (r *CricketMostFoursRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 8, MarketCode: "CK_MOST_FOURS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *CricketMostFoursRule) RequiredDomain() string { return DomainTeamStats }
func (r *CricketMostFoursRule) Timing() string         { return TimingEventEnd }

func (r *CricketMostFoursRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Cricket == nil || (!state.Cricket.MatchCompleted && !state.IsFinished) {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats}
	}
	h, a := state.Cricket.Fours.Home, state.Cricket.Fours.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "TIE" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainTeamStats, EvaluatedAt: time.Now()}
}
