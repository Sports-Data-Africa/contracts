package market

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	// Register Tennis (SportID 3) Rules
	Register(&TennisMatchWinnerRule{})
	Register(&TennisSetWinnerRule{SetNumber: 1, Period: PeriodSet1})
	Register(&TennisSetWinnerRule{SetNumber: 2, Period: PeriodSet2})
	Register(&TennisSetWinnerRule{SetNumber: 3, Period: PeriodSet3})
	Register(&TennisTotalGamesRule{})
	Register(&TennisGameHandicapRule{})
	Register(&TennisSetHandicapRule{})
	Register(&TennisCorrectSetScoreRule{})
}

// ----------------------------------------------------------------------------
// Match Winner (EVENT_END with Retirement / Walkover handling)
// ----------------------------------------------------------------------------
type TennisMatchWinnerRule struct{}

func (r *TennisMatchWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_MATCH_WINNER", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TennisMatchWinnerRule) RequiredDomain() string { return DomainSets }
func (r *TennisMatchWinnerRule) Timing() string         { return TimingEventEnd }

func (r *TennisMatchWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	// Retirement / Walkover handling:
	if state.TennisWalkover {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Match walkover - all match bets void", RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
	if state.TennisRetired && !state.IsFinished {
		// House rule: If player retires before completion of match and winner not determined, match winner voids
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Player retired before match completion - match bets void", RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}

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
// Set Winner (Set 1, Set 2, Set 3) (PERIOD_END)
// ----------------------------------------------------------------------------
type TennisSetWinnerRule struct {
	SetNumber int
	Period    string
}

func (r *TennisSetWinnerRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_SET_WINNER", Period: r.Period, RuleVersion: 1}
}
func (r *TennisSetWinnerRule) RequiredDomain() string { return DomainGames }
func (r *TennisSetWinnerRule) Timing() string         { return TimingPeriodEnd }

func (r *TennisSetWinnerRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.TennisWalkover {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "Walkover", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}

	var set TennisSetData
	found := false
	for _, s := range state.TennisSets {
		if s.SetNumber == r.SetNumber {
			set = s
			found = true
			break
		}
	}

	if !found {
		if state.TennisRetired {
			return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "Player retired before set played", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainGames}
	}

	if !set.IsFinished {
		if state.TennisRetired {
			return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "Player retired during set", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainGames}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1" || sel == "HOME") && set.HomeGames > set.AwayGames ||
		(sel == "2" || sel == "AWAY") && set.AwayGames > set.HomeGames

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Games (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type TennisTotalGamesRule struct{}

func (r *TennisTotalGamesRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_TOTAL_GAMES", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TennisTotalGamesRule) RequiredDomain() string { return DomainGames }
func (r *TennisTotalGamesRule) Timing() string         { return TimingInstantIrreversible }

func (r *TennisTotalGamesRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	totalGames := 0
	for _, s := range state.TennisSets {
		totalGames += s.HomeGames + s.AwayGames
	}

	line := 21.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	// Instant Irreversible: If total games already exceeded line, Over WON immediately!
	if sel == "OVER" && float64(totalGames) > line {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Tennis total games (%d) exceeded line (%.1f)", totalGames, line),
			RequiredDomain: DomainGames,
			EvaluatedAt:    time.Now(),
		}
	}
	if sel == "UNDER" && float64(totalGames) > line {
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Tennis total games (%d) exceeded line (%.1f)", totalGames, line),
			RequiredDomain: DomainGames,
			EvaluatedAt:    time.Now(),
		}
	}

	// Retirement check for unresolved totals:
	if state.TennisRetired && !state.IsFinished {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Player retired before game threshold resolved", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(totalGames) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(totalGames) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainGames}
}

// ----------------------------------------------------------------------------
// Game Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type TennisGameHandicapRule struct{}

func (r *TennisGameHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_GAME_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TennisGameHandicapRule) RequiredDomain() string { return DomainGames }
func (r *TennisGameHandicapRule) Timing() string         { return TimingEventEnd }

func (r *TennisGameHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.TennisRetired || state.TennisWalkover {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Handicap void on retirement/walkover", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainGames}
	}

	homeGames, awayGames := 0, 0
	for _, s := range state.TennisSets {
		homeGames += s.HomeGames
		awayGames += s.AwayGames
	}

	spread := 0.0
	if bet.Line != nil {
		spread = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	var diff float64
	if sel == "1" || sel == "HOME" {
		diff = (float64(homeGames) + spread) - float64(awayGames)
	} else {
		diff = (float64(awayGames) + spread) - float64(homeGames)
	}

	if diff > 0 {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	} else if diff == 0 {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Game handicap push", RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainGames, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Set Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type TennisSetHandicapRule struct{}

func (r *TennisSetHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_SET_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TennisSetHandicapRule) RequiredDomain() string { return DomainSets }
func (r *TennisSetHandicapRule) Timing() string         { return TimingEventEnd }

func (r *TennisSetHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.TennisRetired || state.TennisWalkover {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Set handicap void on retirement/walkover", RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
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
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Correct Set Score (EVENT_END)
// ----------------------------------------------------------------------------
type TennisCorrectSetScoreRule struct{}

func (r *TennisCorrectSetScoreRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 3, MarketCode: "TN_CORRECT_SET_SCORE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *TennisCorrectSetScoreRule) RequiredDomain() string { return DomainSets }
func (r *TennisCorrectSetScoreRule) Timing() string         { return TimingEventEnd }

func (r *TennisCorrectSetScoreRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.TennisRetired || state.TennisWalkover {
		return SettlementResult{Status: BetStatusVoid, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Correct score void on retirement", RequiredDomain: DomainSets, EvaluatedAt: time.Now()}
	}
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
