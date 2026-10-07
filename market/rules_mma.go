package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register MMA (SportID 6) Rules
	Register(&MMAFightWinnerRule{})
	Register(&MMAMethodOfVictoryRule{})
	Register(&MMATotalRoundsRule{})
	Register(&MMAGoesDistanceRule{})
}

// ----------------------------------------------------------------------------
// Fight Winner (EVENT_END with NO CONTEST / DRAW handling)
// ----------------------------------------------------------------------------
type MMAFightWinnerRule struct{}

func (r *MMAFightWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 6, MarketCode: "MM_FIGHT_WINNER", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *MMAFightWinnerRule) RequiredDomain() string { return DomainRounds }
func (r *MMAFightWinnerRule) Timing() string         { return TimingEventEnd }

func (r *MMAFightWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.MMA == nil || !state.MMA.OfficialResultCertified {
		if state.IsFinished {
			// Fallback to Score if MMA state missing
			h, a := state.Score.Home, state.Score.Away
			sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
			if (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainRounds}
	}

	winner := state.MMA.WinningFighter
	if winner == "NO_CONTEST" {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Fight ruled No Contest (void)", RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}
	if winner == "DRAW" {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Fight ended in Draw (push)", RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1" || sel == "HOME") && winner == "1" || (sel == "2" || sel == "AWAY") && winner == "2"

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Method of Victory (EVENT_END)
// ----------------------------------------------------------------------------
type MMAMethodOfVictoryRule struct{}

func (r *MMAMethodOfVictoryRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 6, MarketCode: "MM_METHOD_OF_VICTORY", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *MMAMethodOfVictoryRule) RequiredDomain() string { return DomainRounds }
func (r *MMAMethodOfVictoryRule) Timing() string         { return TimingEventEnd }

func (r *MMAMethodOfVictoryRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.MMA == nil || !state.MMA.OfficialResultCertified {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainRounds}
	}
	if state.MMA.WinningFighter == "NO_CONTEST" {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "No contest", RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}

	method := strings.ToUpper(strings.TrimSpace(state.MMA.MethodOfVictory))
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := false
	switch sel {
	case "KO", "TKO", "KO/TKO":
		won = method == "KO" || method == "TKO"
	case "SUBMISSION", "SUB":
		won = method == "SUBMISSION"
	case "DECISION", "DEC":
		won = strings.Contains(method, "DECISION")
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Rounds (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type MMATotalRoundsRule struct{}

func (r *MMATotalRoundsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 6, MarketCode: "MM_TOTAL_ROUNDS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *MMATotalRoundsRule) RequiredDomain() string { return DomainRounds }
func (r *MMATotalRoundsRule) Timing() string         { return TimingInstantIrreversible }

func (r *MMATotalRoundsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.MMA == nil {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainRounds}
	}

	completedRounds := float64(state.MMA.CompletedRounds)
	if state.MMA.EndTimeSeconds > 0 {
		completedRounds += float64(state.MMA.EndTimeSeconds) / 300.0 // 5-minute round ratio
	}

	line := 2.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && completedRounds > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("MMA round elapsed (%.2f) exceeded line (%.1f)", completedRounds, line), RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && completedRounds > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("MMA round elapsed (%.2f) exceeded line (%.1f)", completedRounds, line), RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}

	if state.MMA.OfficialResultCertified || state.IsFinished {
		if state.MMA.WinningFighter == "NO_CONTEST" {
			return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "No Contest", RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
		}
		switch sel {
		case "OVER":
			if completedRounds > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
		case "UNDER":
			if completedRounds < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainRounds}
}

// ----------------------------------------------------------------------------
// Fight Goes Distance (EVENT_END)
// ----------------------------------------------------------------------------
type MMAGoesDistanceRule struct{}

func (r *MMAGoesDistanceRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 6, MarketCode: "MM_GOES_DISTANCE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *MMAGoesDistanceRule) RequiredDomain() string { return DomainRounds }
func (r *MMAGoesDistanceRule) Timing() string         { return TimingEventEnd }

func (r *MMAGoesDistanceRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.MMA == nil || !state.MMA.OfficialResultCertified {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainRounds}
	}
	if state.MMA.WinningFighter == "NO_CONTEST" {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "No contest", RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}

	goesDistance := strings.Contains(strings.ToUpper(state.MMA.MethodOfVictory), "DECISION")
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	won := (sel == "YES" && goesDistance) || (sel == "NO" && !goesDistance)
	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainRounds, EvaluatedAt: time.Now()}
}
