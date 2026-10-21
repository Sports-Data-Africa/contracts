package market

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func init() {
	// Register Football (SportID 1) Rules
	Register(&Football1X2Rule{})
	Register(&FootballTotalGoalsRule{Period: PeriodFullTime})
	Register(&FootballTotalGoalsRule{Period: PeriodHalfTime})
	Register(&FootballTotalGoalsRule{Period: PeriodSecondHalf})
	Register(&FootballBTTSRule{})
	Register(&FootballDoubleChanceRule{})
	Register(&FootballDrawNoBetRule{})
	Register(&FootballAsianHandicapRule{})
	Register(&FootballEuropeanHandicapRule{})
	Register(&FootballCorrectScoreRule{})
	Register(&FootballHT1X2Rule{})
	Register(&Football2H1X2Rule{})
	Register(&FootballHTFTRule{})
	Register(&FootballOddEvenRule{})
	Register(&FootballCleanSheetRule{})
	Register(&FootballWinToNilRule{})
	Register(&FootballTeamTotalsRule{Team: "HOME", Period: PeriodFullTime})
	Register(&FootballTeamTotalsRule{Team: "AWAY", Period: PeriodFullTime})
	Register(&FootballCorners1X2Rule{})
	Register(&FootballCornersTotalRule{})
	Register(&FootballCardsTotalRule{})

	// Phase 1 Additional Verified Score-Derived Settlement Rules:
	Register(&FootballHTBTTSRule{})
	Register(&FootballHTDoubleChanceRule{})
	Register(&FootballHTDrawNoBetRule{})
	Register(&FootballHTAsianHandicapRule{})
	Register(&FootballHTEuropeanHandicapRule{})
	Register(&FootballHTCorrectScoreRule{})
	Register(&FootballHTCleanSheetRule{})
	Register(&FootballHTWinToNilRule{})
	Register(&FootballTeamTotalsRule{Team: "HOME", Period: PeriodHalfTime})
	Register(&FootballTeamTotalsRule{Team: "AWAY", Period: PeriodHalfTime})
	Register(&FootballExactGoalsRule{Period: PeriodFullTime})
	Register(&FootballExactGoalsRule{Period: PeriodHalfTime})
	Register(&FootballGoalRangeRule{Period: PeriodFullTime})
	Register(&FootballGoalRangeRule{Period: PeriodHalfTime})
	Register(&FootballTeamExactGoalsRule{Team: "HOME"})
	Register(&FootballTeamExactGoalsRule{Team: "AWAY"})
	Register(&FootballTeamOddEvenRule{Team: "HOME"})
	Register(&FootballTeamOddEvenRule{Team: "AWAY"})
	Register(&FootballHTOddEvenRule{})
	Register(&FootballHighestScoringHalfRule{})
	Register(&FootballTeamHighestHalfRule{Team: "HOME"})
	Register(&FootballTeamHighestHalfRule{Team: "AWAY"})
	Register(&FootballWinBothHalvesRule{})
	Register(&FootballScoreBothHalvesRule{})
	Register(&FootballToWinEitherHalfRule{})
	Register(&FootballMultiGoalsRule{})
	Register(&FootballResultBTTSRule{})
	Register(&FootballResultTotalsRule{})
	Register(&FootballBTTSTotalsRule{})
}

// ----------------------------------------------------------------------------
// 1X2 / Match Result (EVENT_END)
// ----------------------------------------------------------------------------
type Football1X2Rule struct{}

func (r *Football1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_1X2", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *Football1X2Rule) RequiredDomain() string { return DomainScore }
func (r *Football1X2Rule) Timing() string         { return TimingEventEnd }

func (r *Football1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1 := sel == "1" || sel == "HOME"
	isX := sel == "X" || sel == "DRAW"
	is2 := sel == "2" || sel == "AWAY"
	if !is1 && !isX && !is2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 1X2", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	won := is1 && h > a || isX && h == a || is2 && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Total Goals (Over / Under) - INSTANT_IRREVERSIBLE
// ----------------------------------------------------------------------------
type FootballTotalGoalsRule struct {
	Period string
}

func (r *FootballTotalGoalsRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	return MarketRuleKey{SportID: 1, MarketCode: "FB_TOTAL_GOALS", Period: p, RuleVersion: 1}
}
func (r *FootballTotalGoalsRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime || r.Period == PeriodSecondHalf {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *FootballTotalGoalsRule) Timing() string { return TimingInstantIrreversible }

func (r *FootballTotalGoalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var currentGoals int
	var periodDone bool

	switch r.Period {
	case PeriodHalfTime:
		if ht, ok := state.PeriodScores["1H"]; ok {
			currentGoals = ht.Home + ht.Away
		} else if ht, ok := state.PeriodScores["HT"]; ok {
			currentGoals = ht.Home + ht.Away
		} else {
			// Fail-closed: missing HT score must NEVER fallback to full-time score
			return SettlementResult{
				Status:         BetStatusSettlementHold,
				Settled:        false,
				Reason:         "Missing authoritative Half-Time period score data",
				Timing:         r.Timing(),
				RequiredDomain: r.RequiredDomain(),
			}
		}
		periodDone = state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	case PeriodSecondHalf:
		if p2, ok := state.PeriodScores["2H"]; ok {
			currentGoals = p2.Home + p2.Away
		} else {
			// Fail-closed: missing 2H score must NEVER be inferred or subtracted from FT score
			return SettlementResult{
				Status:         BetStatusSettlementHold,
				Settled:        false,
				Reason:         "Missing authoritative 2nd Half period score data",
				Timing:         r.Timing(),
				RequiredDomain: r.RequiredDomain(),
			}
		}
		periodDone = state.IsFinished
	default:
		currentGoals = state.Score.Home + state.Score.Away
		periodDone = state.IsFinished
	}

	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{
			Status:         BetStatusSettlementHold,
			Settled:        false,
			Reason:         "Negative score data detected",
			Timing:         r.Timing(),
			RequiredDomain: r.RequiredDomain(),
		}
	}
	if bet.Line == nil {
		return SettlementResult{
			Status:         BetStatusSettlementHold,
			Settled:        false,
			Reason:         "Missing required line specifier for Total Goals",
			Timing:         r.Timing(),
			RequiredDomain: r.RequiredDomain(),
		}
	}
	line := *bet.Line
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "OVER" && sel != "UNDER" {
		return SettlementResult{
			Status:         BetStatusSettlementHold,
			Settled:        false,
			Reason:         fmt.Sprintf("Unsupported selection code '%s' for Total Goals", bet.SelectionCode),
			Timing:         r.Timing(),
			RequiredDomain: r.RequiredDomain(),
		}
	}

	// 1. Instant Irreversible checks
	switch sel {
	case "OVER":
		if float64(currentGoals) > line {
			// Instant WIN once threshold crossed
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingInstantIrreversible,
				Reason:         fmt.Sprintf("Total goals (%d) exceeded line (%.1f)", currentGoals, line),
				RequiredDomain: r.RequiredDomain(),
				EvaluatedAt:    time.Now(),
			}
		}
	case "UNDER":
		if float64(currentGoals) > line {
			// Instant LOSS once threshold exceeded (can never reverse)
			return SettlementResult{
				Status:         BetStatusLost,
				PayoutFactor:   0.0,
				Settled:        true,
				Timing:         TimingInstantIrreversible,
				Reason:         fmt.Sprintf("Total goals (%d) exceeded under line (%.1f)", currentGoals, line),
				RequiredDomain: r.RequiredDomain(),
				EvaluatedAt:    time.Now(),
			}
		}
	}

	// 2. If period/match completed, evaluate remaining non-busted selections
	if periodDone {
		switch sel {
		case "OVER":
			if float64(currentGoals) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(currentGoals) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(currentGoals) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(currentGoals) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: r.RequiredDomain()}
}

// ----------------------------------------------------------------------------
// Both Teams To Score (BTTS / GG) - INSTANT_IRREVERSIBLE
// ----------------------------------------------------------------------------
type FootballBTTSRule struct{}

func (r *FootballBTTSRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_BTTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballBTTSRule) RequiredDomain() string { return DomainScore }
func (r *FootballBTTSRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballBTTSRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isYes := sel == "YES" || sel == "GG"
	isNo := sel == "NO" || sel == "NG"
	if !isYes && !isNo {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for BTTS", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	bothScored := h >= 1 && a >= 1

	// Instant Irreversible resolution
	if bothScored {
		if isYes {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingInstantIrreversible,
				Reason:         fmt.Sprintf("Both teams confirmed scored (%d-%d)", h, a),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now(),
			}
		}
		if isNo {
			return SettlementResult{
				Status:         BetStatusLost,
				PayoutFactor:   0.0,
				Settled:        true,
				Timing:         TimingInstantIrreversible,
				Reason:         fmt.Sprintf("Both teams confirmed scored (%d-%d)", h, a),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now(),
			}
		}
	}

	// At match end: if not both scored
	if state.IsFinished {
		if isNo {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Double Chance (EVENT_END)
// ----------------------------------------------------------------------------
type FootballDoubleChanceRule struct{}

func (r *FootballDoubleChanceRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_DOUBLE_CHANCE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballDoubleChanceRule) RequiredDomain() string { return DomainScore }
func (r *FootballDoubleChanceRule) Timing() string         { return TimingEventEnd }

func (r *FootballDoubleChanceRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "1X" && sel != "12" && sel != "X2" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Double Chance", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	won := (sel == "1X" && h >= a) ||
		(sel == "12" && h != a) ||
		(sel == "X2" && a >= h)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Draw No Bet (EVENT_END with PUSH)
// ----------------------------------------------------------------------------
type FootballDrawNoBetRule struct{}

func (r *FootballDrawNoBetRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_DNB", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballDrawNoBetRule) RequiredDomain() string { return DomainScore }
func (r *FootballDrawNoBetRule) Timing() string         { return TimingEventEnd }

func (r *FootballDrawNoBetRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1 := sel == "1" || sel == "HOME"
	is2 := sel == "2" || sel == "AWAY"
	if !is1 && !is2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Draw No Bet", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Match ended in a draw (DNB push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if is1 && h > a || is2 && a > h {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Asian Handicap (EVENT_END - Exact Integer Quarters Arithmetic)
// ----------------------------------------------------------------------------
type FootballAsianHandicapRule struct{}

func (r *FootballAsianHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_ASIAN_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballAsianHandicapRule) RequiredDomain() string { return DomainScore }
func (r *FootballAsianHandicapRule) Timing() string         { return TimingEventEnd }

func (r *FootballAsianHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	if bet.LineQuarters == nil && bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for Asian Handicap", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "1" && sel != "HOME" && sel != "2" && sel != "AWAY" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Asian Handicap", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	lineQuarters := 0
	if bet.LineQuarters != nil {
		lineQuarters = *bet.LineQuarters
	} else if bet.Line != nil {
		lineQuarters = FloatToLineQuarters(*bet.Line)
	}

	outcome, factor := EvaluateAsianHandicap(state.Score.Home, state.Score.Away, bet.SelectionCode, lineQuarters)

	var status string
	switch outcome {
	case AsianWin:
		status = BetStatusWon
	case AsianHalfWin:
		status = BetStatusHalfWin
	case AsianPush:
		status = BetStatusPush
	case AsianHalfLoss:
		status = BetStatusHalfLoss
	case AsianLoss:
		status = BetStatusLost
	}

	return SettlementResult{
		Status:         status,
		PayoutFactor:   factor,
		Settled:        true,
		Timing:         TimingEventEnd,
		Reason:         fmt.Sprintf("Asian Handicap (%d quarters): outcome=%s", lineQuarters, outcome),
		RequiredDomain: DomainScore,
		EvaluatedAt:    time.Now(),
	}
}

// ----------------------------------------------------------------------------
// European Handicap / 3-Way Handicap (EVENT_END)
// ----------------------------------------------------------------------------
type FootballEuropeanHandicapRule struct{}

func (r *FootballEuropeanHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_EUROPEAN_HANDICAP", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballEuropeanHandicapRule) RequiredDomain() string { return DomainScore }
func (r *FootballEuropeanHandicapRule) Timing() string         { return TimingEventEnd }

func (r *FootballEuropeanHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	if bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for European Handicap", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "1" && sel != "HOME" && sel != "X" && sel != "DRAW" && sel != "2" && sel != "AWAY" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for European Handicap", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	handicap := int(*bet.Line)
	adjHome := state.Score.Home + handicap
	away := state.Score.Away

	won := (sel == "1" || sel == "HOME") && adjHome > away ||
		(sel == "X" || sel == "DRAW") && adjHome == away ||
		(sel == "2" || sel == "AWAY") && away > adjHome

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Correct Score (EVENT_END)
// ----------------------------------------------------------------------------
type FootballCorrectScoreRule struct{}

func (r *FootballCorrectScoreRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_CORRECT_SCORE", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballCorrectScoreRule) RequiredDomain() string { return DomainScore }
func (r *FootballCorrectScoreRule) Timing() string         { return TimingEventEnd }

func (r *FootballCorrectScoreRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	targetScore := strings.TrimSpace(bet.SelectionCode)
	actualScore := fmt.Sprintf("%d:%d", state.Score.Home, state.Score.Away)
	actualScoreHyphen := fmt.Sprintf("%d-%d", state.Score.Home, state.Score.Away)

	if targetScore == actualScore || targetScore == actualScoreHyphen {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Half Time Result (PERIOD_END)
// ----------------------------------------------------------------------------
type FootballHT1X2Rule struct{}

func (r *FootballHT1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_1X2", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHT1X2Rule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHT1X2Rule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHT1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	htScore, ok := state.PeriodScores["1H"]
	if !ok {
		htScore, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished

	if !ok || !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if htScore.Home < 0 || htScore.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1 := sel == "1" || sel == "HOME"
	isX := sel == "X" || sel == "DRAW"
	is2 := sel == "2" || sel == "AWAY"
	if !is1 && !isX && !is2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for HT 1X2", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	h, a := htScore.Home, htScore.Away
	won := is1 && h > a || isX && h == a || is2 && a > h

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 2nd Half Result (PERIOD_END)
// ----------------------------------------------------------------------------
type Football2H1X2Rule struct{}

func (r *Football2H1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_2H_1X2", Period: PeriodSecondHalf, RuleVersion: 1}
}
func (r *Football2H1X2Rule) RequiredDomain() string { return DomainPeriodScore }
func (r *Football2H1X2Rule) Timing() string         { return TimingPeriodEnd }

func (r *Football2H1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	h2, ok := state.PeriodScores["2H"]
	if !ok {
		ht, okHT := state.PeriodScores["1H"]
		if !okHT {
			ht, okHT = state.PeriodScores["HT"]
		}
		if okHT {
			if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
				return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
			}
			h2 = ScoreData{Home: state.Score.Home - ht.Home, Away: state.Score.Away - ht.Away}
		} else {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score to derive 2H result", RequiredDomain: DomainPeriodScore}
		}
	}
	if h2.Home < 0 || h2.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative 2H score data detected", RequiredDomain: DomainPeriodScore}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1 := sel == "1" || sel == "HOME"
	isX := sel == "X" || sel == "DRAW"
	is2 := sel == "2" || sel == "AWAY"
	if !is1 && !isX && !is2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 2H 1X2", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := is1 && h2.Home > h2.Away ||
		isX && h2.Home == h2.Away ||
		is2 && h2.Away > h2.Home

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Half Time / Full Time (HT/FT) (EVENT_END)
// ----------------------------------------------------------------------------
type FootballHTFTRule struct{}

func (r *FootballHTFTRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_FT", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballHTFTRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTFTRule) Timing() string         { return TimingEventEnd }

func (r *FootballHTFTRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing HT score for HT/FT settlement", RequiredDomain: DomainPeriodScore}
	}
	if ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative HT score data detected", RequiredDomain: DomainPeriodScore}
	}

	htOutcome := "X"
	if ht.Home > ht.Away {
		htOutcome = "1"
	} else if ht.Away > ht.Home {
		htOutcome = "2"
	}

	ftOutcome := "X"
	if state.Score.Home > state.Score.Away {
		ftOutcome = "1"
	} else if state.Score.Away > state.Score.Home {
		ftOutcome = "2"
	}

	actualCombo := fmt.Sprintf("%s/%s", htOutcome, ftOutcome)
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	sel = strings.ReplaceAll(sel, "-", "/")

	if sel == actualCombo {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Odd / Even Goals (EVENT_END)
// ----------------------------------------------------------------------------
type FootballOddEvenRule struct{}

func (r *FootballOddEvenRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_ODD_EVEN", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballOddEvenRule) RequiredDomain() string { return DomainScore }
func (r *FootballOddEvenRule) Timing() string         { return TimingEventEnd }

func (r *FootballOddEvenRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	total := state.Score.Home + state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "ODD" && sel != "EVEN" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Odd/Even", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	isOdd := total%2 != 0
	won := (sel == "ODD" && isOdd) || (sel == "EVEN" && !isOdd)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Clean Sheet (EVENT_END or INSTANT_IRREVERSIBLE for Lost)
// ----------------------------------------------------------------------------
type FootballCleanSheetRule struct{}

func (r *FootballCleanSheetRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_CLEAN_SHEET", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballCleanSheetRule) RequiredDomain() string { return DomainScore }
func (r *FootballCleanSheetRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballCleanSheetRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHomeCleanSheet := strings.Contains(sel, "HOME") || sel == "1"
	isAwayCleanSheet := strings.Contains(sel, "AWAY") || sel == "2"
	if !isHomeCleanSheet && !isAwayCleanSheet {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Clean Sheet", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	if isHomeCleanSheet {
		// Home clean sheet means Away scores 0. If Away scores >= 1, clean sheet is instantly LOST!
		if state.Score.Away >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Away team scored, clean sheet lost", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if state.IsFinished {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	} else if isAwayCleanSheet {
		// Away clean sheet: Home scores 0. If Home scores >= 1, clean sheet is instantly LOST!
		if state.Score.Home >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Home team scored, clean sheet lost", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if state.IsFinished {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Win To Nil (EVENT_END or INSTANT_IRREVERSIBLE for Lost)
// ----------------------------------------------------------------------------
type FootballWinToNilRule struct{}

func (r *FootballWinToNilRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_WIN_TO_NIL", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballWinToNilRule) RequiredDomain() string { return DomainScore }
func (r *FootballWinToNilRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballWinToNilRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "1" || strings.Contains(sel, "HOME")
	isAway := sel == "2" || strings.Contains(sel, "AWAY")
	if !isHome && !isAway {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Win To Nil", bet.SelectionCode), RequiredDomain: DomainScore}
	}

	if isHome {
		// If Away scores, Home cannot Win to Nil -> Instant LOSS
		if state.Score.Away >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Opponent scored, Win To Nil lost", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if state.IsFinished {
			if state.Score.Home > 0 && state.Score.Away == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	} else if isAway {
		// Away win to nil: If Home scores, Away cannot Win to Nil -> Instant LOSS
		if state.Score.Home >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Opponent scored, Win To Nil lost", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if state.IsFinished {
			if state.Score.Away > 0 && state.Score.Home == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
}

// ----------------------------------------------------------------------------
// Team Totals (INSTANT_IRREVERSIBLE)
// ----------------------------------------------------------------------------
type FootballTeamTotalsRule struct {
	Team   string // HOME or AWAY
	Period string // FT or HT
}

func (r *FootballTeamTotalsRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	code := "FB_HOME_TOTALS"
	if r.Team == "AWAY" {
		code = "FB_AWAY_TOTALS"
	}
	if p == PeriodHalfTime {
		if r.Team == "HOME" {
			code = "FB_HOME_TOTALS_HT"
		} else {
			code = "FB_AWAY_TOTALS_HT"
		}
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: p, RuleVersion: 1}
}
func (r *FootballTeamTotalsRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *FootballTeamTotalsRule) Timing() string { return TimingInstantIrreversible }

func (r *FootballTeamTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var goals int
	var isDone bool

	if r.Period == PeriodHalfTime {
		ht, ok := state.PeriodScores["1H"]
		if !ok {
			ht, ok = state.PeriodScores["HT"]
		}
		isDone = state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
		if ok {
			if r.Team == "AWAY" {
				goals = ht.Away
			} else {
				goals = ht.Home
			}
		} else if isDone {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Team Totals HT settlement", RequiredDomain: DomainPeriodScore}
		}
	} else {
		isDone = state.IsFinished
		if r.Team == "AWAY" {
			goals = state.Score.Away
		} else {
			goals = state.Score.Home
		}
	}

	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: r.RequiredDomain()}
	}
	if bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for Team Totals", RequiredDomain: r.RequiredDomain()}
	}
	line := *bet.Line
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "OVER" && sel != "UNDER" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Team Totals", bet.SelectionCode), RequiredDomain: r.RequiredDomain()}
	}

	switch sel {
	case "OVER":
		if float64(goals) > line {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Team goals (%d) exceeded line (%.1f)", goals, line), RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	case "UNDER":
		if float64(goals) > line {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Team goals (%d) exceeded under line (%.1f)", goals, line), RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	}

	if isDone {
		switch sel {
		case "OVER":
			if float64(goals) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(goals) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(goals) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			} else if float64(goals) == line {
				return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: r.RequiredDomain()}
}

// ----------------------------------------------------------------------------
// Corners 1X2 (EVENT_END - DomainCorners)
// ----------------------------------------------------------------------------
type FootballCorners1X2Rule struct{}

func (r *FootballCorners1X2Rule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_CORNERS_1X2", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballCorners1X2Rule) RequiredDomain() string { return DomainCorners }
func (r *FootballCorners1X2Rule) Timing() string         { return TimingEventEnd }

func (r *FootballCorners1X2Rule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainCorners}
	}
	corners, ok := state.Corners["FT"]
	if !ok {
		return SettlementResult{Status: BetStatusManualReview, Settled: false, Reason: "Corners FT data not available", RequiredDomain: DomainCorners}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1" || sel == "HOME") && corners.Home > corners.Away ||
		(sel == "X" || sel == "DRAW") && corners.Home == corners.Away ||
		(sel == "2" || sel == "AWAY") && corners.Away > corners.Home

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Corners Total (INSTANT_IRREVERSIBLE - DomainCorners)
// ----------------------------------------------------------------------------
type FootballCornersTotalRule struct{}

func (r *FootballCornersTotalRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_CORNERS_TOTAL", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballCornersTotalRule) RequiredDomain() string { return DomainCorners }
func (r *FootballCornersTotalRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballCornersTotalRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	corners, ok := state.Corners["FT"]
	if !ok {
		if state.IsFinished {
			return SettlementResult{Status: BetStatusManualReview, Settled: false, Reason: "Corners data not available", RequiredDomain: DomainCorners}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainCorners}
	}

	total := corners.Total()
	line := 9.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(total) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Corners total (%d) crossed line (%.1f)", total, line), RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(total) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Corners total (%d) exceeded line (%.1f)", total, line), RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(total) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(total) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCorners, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainCorners}
}

// ----------------------------------------------------------------------------
// Cards Total (INSTANT_IRREVERSIBLE - DomainCards)
// ----------------------------------------------------------------------------
type FootballCardsTotalRule struct{}

func (r *FootballCardsTotalRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_CARDS_TOTAL", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballCardsTotalRule) RequiredDomain() string { return DomainCards }
func (r *FootballCardsTotalRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballCardsTotalRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	cards, ok := state.Cards["FT"]
	if !ok {
		if state.IsFinished {
			return SettlementResult{Status: BetStatusManualReview, Settled: false, Reason: "Cards data not available", RequiredDomain: DomainCards}
		}
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainCards}
	}

	// Booking yellow cards total
	total := cards.HomeYellow + cards.AwayYellow + (cards.HomeRed+cards.AwayRed)*2
	line := 3.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if sel == "OVER" && float64(total) > line {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Cards total points (%d) exceeded line (%.1f)", total, line), RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
	}
	if sel == "UNDER" && float64(total) > line {
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Cards total points (%d) exceeded line (%.1f)", total, line), RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(total) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(total) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainCards, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainCards}
}

// ----------------------------------------------------------------------------
// 1st Half Both Teams To Score (FB_HT_BTTS)
// ----------------------------------------------------------------------------
type FootballHTBTTSRule struct{}

func (r *FootballHTBTTSRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_BTTS", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTBTTSRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTBTTSRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballHTBTTSRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if ok && ht.Home >= 1 && ht.Away >= 1 {
		if sel == "YES" || sel == "GG" {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Both teams scored in 1H (%d-%d)", ht.Home, ht.Away), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
		if sel == "NO" || sel == "NG" {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Both teams scored in 1H (%d-%d)", ht.Home, ht.Away), RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if isHTDone {
		if !ok {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT BTTS settlement", RequiredDomain: DomainPeriodScore}
		}
		if ht.Home >= 1 && ht.Away >= 1 {
			if sel == "YES" || sel == "GG" {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
		if sel == "NO" || sel == "NG" {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
		return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPeriodScore}
}

// ----------------------------------------------------------------------------
// 1st Half Double Chance (FB_HT_DOUBLE_CHANCE)
// ----------------------------------------------------------------------------
type FootballHTDoubleChanceRule struct{}

func (r *FootballHTDoubleChanceRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_DOUBLE_CHANCE", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTDoubleChanceRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTDoubleChanceRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTDoubleChanceRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Double Chance settlement", RequiredDomain: DomainPeriodScore}
	}

	h, a := ht.Home, ht.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1X" && h >= a) ||
		(sel == "12" && h != a) ||
		(sel == "X2" && a >= h)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Draw No Bet (FB_HT_DNB)
// ----------------------------------------------------------------------------
type FootballHTDrawNoBetRule struct{}

func (r *FootballHTDrawNoBetRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_DNB", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTDrawNoBetRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTDrawNoBetRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTDrawNoBetRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT DNB settlement", RequiredDomain: DomainPeriodScore}
	}

	h, a := ht.Home, ht.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, Reason: "1st Half ended in a draw (push)", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	if (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Asian Handicap (FB_HT_ASIAN_HANDICAP)
// ----------------------------------------------------------------------------
type FootballHTAsianHandicapRule struct{}

func (r *FootballHTAsianHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_ASIAN_HANDICAP", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTAsianHandicapRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTAsianHandicapRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTAsianHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Asian Handicap settlement", RequiredDomain: DomainPeriodScore}
	}
	if ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if bet.LineQuarters == nil && bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for 1st Half Asian Handicap", RequiredDomain: DomainPeriodScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "1" && sel != "HOME" && sel != "2" && sel != "AWAY" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 1st Half Asian Handicap", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	lineQuarters := 0
	if bet.LineQuarters != nil {
		lineQuarters = *bet.LineQuarters
	} else if bet.Line != nil {
		lineQuarters = FloatToLineQuarters(*bet.Line)
	}

	outcome, factor := EvaluateAsianHandicap(ht.Home, ht.Away, bet.SelectionCode, lineQuarters)
	var status string
	switch outcome {
	case AsianWin:
		status = BetStatusWon
	case AsianHalfWin:
		status = BetStatusHalfWin
	case AsianPush:
		status = BetStatusPush
	case AsianHalfLoss:
		status = BetStatusHalfLoss
	case AsianLoss:
		status = BetStatusLost
	}

	return SettlementResult{
		Status:         status,
		PayoutFactor:   factor,
		Settled:        true,
		Timing:         TimingPeriodEnd,
		Reason:         fmt.Sprintf("HT Asian Handicap (%d quarters): outcome=%s", lineQuarters, outcome),
		RequiredDomain: DomainPeriodScore,
		EvaluatedAt:    time.Now(),
	}
}

// ----------------------------------------------------------------------------
// 1st Half European Handicap (FB_HT_EUROPEAN_HCP)
// ----------------------------------------------------------------------------
type FootballHTEuropeanHandicapRule struct{}

func (r *FootballHTEuropeanHandicapRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_EUROPEAN_HCP", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTEuropeanHandicapRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTEuropeanHandicapRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTEuropeanHandicapRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT European Handicap settlement", RequiredDomain: DomainPeriodScore}
	}
	if ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for 1st Half European Handicap", RequiredDomain: DomainPeriodScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	if sel != "1" && sel != "HOME" && sel != "X" && sel != "DRAW" && sel != "2" && sel != "AWAY" {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 1st Half European Handicap", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	handicap := int(*bet.Line)
	adjHome := ht.Home + handicap
	away := ht.Away

	won := (sel == "1" || sel == "HOME") && adjHome > away ||
		(sel == "X" || sel == "DRAW") && adjHome == away ||
		(sel == "2" || sel == "AWAY") && away > adjHome

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Correct Score (FB_HT_CORRECT_SCORE)
// ----------------------------------------------------------------------------
type FootballHTCorrectScoreRule struct{}

func (r *FootballHTCorrectScoreRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_CORRECT_SCORE", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTCorrectScoreRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTCorrectScoreRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTCorrectScoreRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Correct Score settlement", RequiredDomain: DomainPeriodScore}
	}
	if ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}

	targetScore := strings.TrimSpace(bet.SelectionCode)
	actualColon := fmt.Sprintf("%d:%d", ht.Home, ht.Away)
	actualHyphen := fmt.Sprintf("%d-%d", ht.Home, ht.Away)

	if targetScore == actualColon || targetScore == actualHyphen {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Clean Sheet (FB_HT_CLEAN_SHEET)
// ----------------------------------------------------------------------------
type FootballHTCleanSheetRule struct{}

func (r *FootballHTCleanSheetRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_CLEAN_SHEET", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTCleanSheetRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTCleanSheetRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballHTCleanSheetRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if ok && (ht.Home < 0 || ht.Away < 0) {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHomeCleanSheet := strings.Contains(sel, "HOME") || sel == "1"
	isAwayCleanSheet := strings.Contains(sel, "AWAY") || sel == "2"
	if !isHomeCleanSheet && !isAwayCleanSheet {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 1st Half Clean Sheet", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	if isHomeCleanSheet {
		if ok && ht.Away >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Away team scored in 1H, Home HT clean sheet lost", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	} else if isAwayCleanSheet {
		if ok && ht.Home >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Home team scored in 1H, Away HT clean sheet lost", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if isHTDone {
		if !ok {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Clean Sheet settlement", RequiredDomain: DomainPeriodScore}
		}
		if isHomeCleanSheet {
			if ht.Away == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		} else if isAwayCleanSheet {
			if ht.Home == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPeriodScore}
}

// ----------------------------------------------------------------------------
// 1st Half Win To Nil (FB_HT_WIN_TO_NIL)
// ----------------------------------------------------------------------------
type FootballHTWinToNilRule struct{}

func (r *FootballHTWinToNilRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_WIN_TO_NIL", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTWinToNilRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTWinToNilRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballHTWinToNilRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if ok && (ht.Home < 0 || ht.Away < 0) {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "1" || strings.Contains(sel, "HOME")
	isAway := sel == "2" || strings.Contains(sel, "AWAY")
	if !isHome && !isAway {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for 1st Half Win To Nil", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	if isHome {
		if ok && ht.Away >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Opponent scored in 1H, Home HT Win To Nil lost", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	} else if isAway {
		if ok && ht.Home >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Opponent scored in 1H, Away HT Win To Nil lost", RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if isHTDone {
		if !ok {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Win To Nil settlement", RequiredDomain: DomainPeriodScore}
		}
		if isHome {
			if ht.Home > 0 && ht.Away == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		} else if isAway {
			if ht.Away > 0 && ht.Home == 0 {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainPeriodScore}
}

// ----------------------------------------------------------------------------
// Exact Goals (FT & HT)
// ----------------------------------------------------------------------------
type FootballExactGoalsRule struct {
	Period string
}

func (r *FootballExactGoalsRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	code := "FB_EXACT_GOALS"
	if p == PeriodHalfTime {
		code = "FB_EXACT_GOALS_HT"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: p, RuleVersion: 1}
}
func (r *FootballExactGoalsRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *FootballExactGoalsRule) Timing() string { return TimingEventEnd }

func (r *FootballExactGoalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var total int
	if r.Period == PeriodHalfTime {
		ht, ok := state.PeriodScores["1H"]
		if !ok {
			ht, ok = state.PeriodScores["HT"]
		}
		isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
		if !isHTDone {
			return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
		}
		if !ok {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Exact Goals HT settlement", RequiredDomain: DomainPeriodScore}
		}
		total = ht.Home + ht.Away
	} else {
		if !state.IsFinished {
			return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
		}
		total = state.Score.Home + state.Score.Away
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := false
	if strings.HasSuffix(sel, "+") || strings.HasSuffix(sel, "_PLUS") {
		clean := strings.TrimSuffix(strings.TrimSuffix(sel, "+"), "_PLUS")
		if threshold, err := strconv.Atoi(clean); err == nil {
			won = total >= threshold
		}
	} else {
		if val, err := strconv.Atoi(sel); err == nil {
			won = total == val
		}
	}

	timing := TimingEventEnd
	if r.Period == PeriodHalfTime {
		timing = TimingPeriodEnd
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: timing, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: timing, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Goal Range (FT & HT)
// ----------------------------------------------------------------------------
type FootballGoalRangeRule struct {
	Period string
}

func (r *FootballGoalRangeRule) Key() MarketRuleKey {
	p := r.Period
	if p == "" {
		p = PeriodFullTime
	}
	code := "FB_GOAL_RANGE"
	if p == PeriodHalfTime {
		code = "FB_GOAL_RANGE_HT"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: p, RuleVersion: 1}
}
func (r *FootballGoalRangeRule) RequiredDomain() string {
	if r.Period == PeriodHalfTime {
		return DomainPeriodScore
	}
	return DomainScore
}
func (r *FootballGoalRangeRule) Timing() string { return TimingEventEnd }

func (r *FootballGoalRangeRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	var total int
	if r.Period == PeriodHalfTime {
		ht, ok := state.PeriodScores["1H"]
		if !ok {
			ht, ok = state.PeriodScores["HT"]
		}
		isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
		if !isHTDone {
			return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
		}
		if !ok {
			return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Goal Range HT settlement", RequiredDomain: DomainPeriodScore}
		}
		total = ht.Home + ht.Away
	} else {
		if !state.IsFinished {
			return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
		}
		total = state.Score.Home + state.Score.Away
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := false

	if strings.Contains(sel, "-") {
		parts := strings.Split(sel, "-")
		if len(parts) == 2 {
			min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				won = total >= min && total <= max
			}
		}
	} else if strings.HasSuffix(sel, "+") || strings.HasSuffix(sel, "_PLUS") {
		clean := strings.TrimSuffix(strings.TrimSuffix(sel, "+"), "_PLUS")
		if threshold, err := strconv.Atoi(clean); err == nil {
			won = total >= threshold
		}
	} else {
		if val, err := strconv.Atoi(sel); err == nil {
			won = total == val
		}
	}

	timing := TimingEventEnd
	if r.Period == PeriodHalfTime {
		timing = TimingPeriodEnd
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: timing, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: timing, RequiredDomain: r.RequiredDomain(), EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Team Exact Goals (HOME & AWAY)
// ----------------------------------------------------------------------------
type FootballTeamExactGoalsRule struct {
	Team string // HOME or AWAY
}

func (r *FootballTeamExactGoalsRule) Key() MarketRuleKey {
	code := "FB_HOME_EXACT_GOALS"
	if r.Team == "AWAY" {
		code = "FB_AWAY_EXACT_GOALS"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballTeamExactGoalsRule) RequiredDomain() string { return DomainScore }
func (r *FootballTeamExactGoalsRule) Timing() string         { return TimingEventEnd }

func (r *FootballTeamExactGoalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	goals := state.Score.Home
	if r.Team == "AWAY" {
		goals = state.Score.Away
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := false
	if strings.HasSuffix(sel, "+") || strings.HasSuffix(sel, "_PLUS") {
		clean := strings.TrimSuffix(strings.TrimSuffix(sel, "+"), "_PLUS")
		if threshold, err := strconv.Atoi(clean); err == nil {
			won = goals >= threshold
		}
	} else {
		if val, err := strconv.Atoi(sel); err == nil {
			won = goals == val
		}
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Team Odd/Even Goals (HOME & AWAY)
// ----------------------------------------------------------------------------
type FootballTeamOddEvenRule struct {
	Team string // HOME or AWAY
}

func (r *FootballTeamOddEvenRule) Key() MarketRuleKey {
	code := "FB_HOME_ODD_EVEN"
	if r.Team == "AWAY" {
		code = "FB_AWAY_ODD_EVEN"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballTeamOddEvenRule) RequiredDomain() string { return DomainScore }
func (r *FootballTeamOddEvenRule) Timing() string         { return TimingEventEnd }

func (r *FootballTeamOddEvenRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	goals := state.Score.Home
	if r.Team == "AWAY" {
		goals = state.Score.Away
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isOdd := goals%2 != 0
	won := (sel == "ODD" && isOdd) || (sel == "EVEN" && !isOdd)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// 1st Half Odd/Even (FB_HT_ODD_EVEN)
// ----------------------------------------------------------------------------
type FootballHTOddEvenRule struct{}

func (r *FootballHTOddEvenRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HT_ODD_EVEN", Period: PeriodHalfTime, RuleVersion: 1}
}
func (r *FootballHTOddEvenRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHTOddEvenRule) Timing() string         { return TimingPeriodEnd }

func (r *FootballHTOddEvenRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	isHTDone := state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	if !isHTDone {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore}
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for HT Odd/Even settlement", RequiredDomain: DomainPeriodScore}
	}

	total := ht.Home + ht.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isOdd := total%2 != 0
	won := (sel == "ODD" && isOdd) || (sel == "EVEN" && !isOdd)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingPeriodEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Highest Scoring Half (FB_HIGHEST_SCORING_HALF)
// ----------------------------------------------------------------------------
type FootballHighestScoringHalfRule struct{}

func (r *FootballHighestScoringHalfRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_HIGHEST_SCORING_HALF", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballHighestScoringHalfRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballHighestScoringHalfRule) Timing() string         { return TimingEventEnd }

func (r *FootballHighestScoringHalfRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Highest Scoring Half settlement", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 || ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
	}

	g1 := ht.Home + ht.Away
	g2 := (state.Score.Home - ht.Home) + (state.Score.Away - ht.Away)

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1st := sel == "1ST" || sel == "1" || sel == "1H"
	is2nd := sel == "2ND" || sel == "2" || sel == "2H"
	isEqual := sel == "EQUAL" || sel == "X" || sel == "DRAW" || sel == "TIE"
	if !is1st && !is2nd && !isEqual {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Highest Scoring Half", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := (is1st && g1 > g2) || (is2nd && g2 > g1) || (isEqual && g1 == g2)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Team Highest Scoring Half (FB_HOME_HIGHEST_HALF & FB_AWAY_HIGHEST_HALF)
// ----------------------------------------------------------------------------
type FootballTeamHighestHalfRule struct {
	Team string // HOME or AWAY
}

func (r *FootballTeamHighestHalfRule) Key() MarketRuleKey {
	code := "FB_HOME_HIGHEST_HALF"
	if r.Team == "AWAY" {
		code = "FB_AWAY_HIGHEST_HALF"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballTeamHighestHalfRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballTeamHighestHalfRule) Timing() string         { return TimingEventEnd }

func (r *FootballTeamHighestHalfRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Team Highest Half settlement", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 || ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
	}

	var g1, g2 int
	if r.Team == "AWAY" {
		g1 = ht.Away
		g2 = state.Score.Away - ht.Away
	} else {
		g1 = ht.Home
		g2 = state.Score.Home - ht.Home
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	is1st := sel == "1ST" || sel == "1" || sel == "1H"
	is2nd := sel == "2ND" || sel == "2" || sel == "2H"
	isEqual := sel == "EQUAL" || sel == "X" || sel == "DRAW" || sel == "TIE"
	if !is1st && !is2nd && !isEqual {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Team Highest Half", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := (is1st && g1 > g2) || (is2nd && g2 > g1) || (isEqual && g1 == g2)

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Win Both Halves (FB_WIN_BOTH_HALVES)
// ----------------------------------------------------------------------------
type FootballWinBothHalvesRule struct{}

func (r *FootballWinBothHalvesRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_WIN_BOTH_HALVES", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballWinBothHalvesRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballWinBothHalvesRule) Timing() string         { return TimingEventEnd }

func (r *FootballWinBothHalvesRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Win Both Halves settlement", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 || ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
	}

	h1, a1 := ht.Home, ht.Away
	h2, a2 := state.Score.Home-ht.Home, state.Score.Away-ht.Away

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "HOME" || sel == "1"
	isAway := sel == "AWAY" || sel == "2"
	if !isHome && !isAway {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Win Both Halves", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := false
	if isHome {
		won = h1 > a1 && h2 > a2
	} else if isAway {
		won = a1 > h1 && a2 > h2
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Score In Both Halves (FB_SCORE_BOTH_HALVES)
// ----------------------------------------------------------------------------
type FootballScoreBothHalvesRule struct{}

func (r *FootballScoreBothHalvesRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_SCORE_BOTH_HALVES", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballScoreBothHalvesRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballScoreBothHalvesRule) Timing() string         { return TimingEventEnd }

func (r *FootballScoreBothHalvesRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for Score Both Halves settlement", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 || ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "HOME" || sel == "1"
	isAway := sel == "AWAY" || sel == "2"
	if !isHome && !isAway {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for Score Both Halves", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := false
	if isHome {
		won = ht.Home > 0 && (state.Score.Home-ht.Home) > 0
	} else if isAway {
		won = ht.Away > 0 && (state.Score.Away-ht.Away) > 0
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// To Win Either Half (FB_TO_WIN_EITHER_HALF)
// ----------------------------------------------------------------------------
type FootballToWinEitherHalfRule struct{}

func (r *FootballToWinEitherHalfRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_TO_WIN_EITHER_HALF", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballToWinEitherHalfRule) RequiredDomain() string { return DomainPeriodScore }
func (r *FootballToWinEitherHalfRule) Timing() string         { return TimingEventEnd }

func (r *FootballToWinEitherHalfRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore}
	}
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing 1H score for To Win Either Half settlement", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 || ht.Home < 0 || ht.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainPeriodScore}
	}
	if state.Score.Home < ht.Home || state.Score.Away < ht.Away {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Inconsistent score data: Full-time score is less than half-time score", RequiredDomain: DomainPeriodScore}
	}

	h1, a1 := ht.Home, ht.Away
	h2, a2 := state.Score.Home-ht.Home, state.Score.Away-ht.Away

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "HOME" || sel == "1"
	isAway := sel == "AWAY" || sel == "2"
	if !isHome && !isAway {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Unsupported selection code '%s' for To Win Either Half", bet.SelectionCode), RequiredDomain: DomainPeriodScore}
	}

	won := false
	if isHome {
		won = h1 > a1 || h2 > a2
	} else if isAway {
		won = a1 > h1 || a2 > h2
	}

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainPeriodScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Multi Goals (FB_MULTI_GOALS)
// ----------------------------------------------------------------------------
type FootballMultiGoalsRule struct{}

func (r *FootballMultiGoalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_MULTI_GOALS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballMultiGoalsRule) RequiredDomain() string { return DomainScore }
func (r *FootballMultiGoalsRule) Timing() string         { return TimingEventEnd }

func (r *FootballMultiGoalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	total := state.Score.Home + state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	parts := strings.Split(sel, "-")
	if len(parts) != 2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Invalid multi-goals selection code '%s': expected format MIN-MAX", bet.SelectionCode), RequiredDomain: DomainScore}
	}
	min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || min < 0 || min > max {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Invalid multi-goals bounds in selection code '%s'", bet.SelectionCode), RequiredDomain: DomainScore}
	}
	won := total >= min && total <= max

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Match Result and Both Teams To Score (FB_RESULT_BTTS)
// ----------------------------------------------------------------------------
type FootballResultBTTSRule struct{}

func (r *FootballResultBTTSRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_RESULT_BTTS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballResultBTTSRule) RequiredDomain() string { return DomainScore }
func (r *FootballResultBTTSRule) Timing() string         { return TimingEventEnd }

func (r *FootballResultBTTSRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	h, a := state.Score.Home, state.Score.Away
	resultOutcome := "X"
	if h > a {
		resultOutcome = "1"
	} else if a > h {
		resultOutcome = "2"
	}

	bttsOutcome := "NO"
	if h >= 1 && a >= 1 {
		bttsOutcome = "YES"
	}

	cleanSel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	cleanSel = strings.ReplaceAll(cleanSel, "AND", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "+", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "/", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "HOME", "1")
	cleanSel = strings.ReplaceAll(cleanSel, "DRAW", "X")
	cleanSel = strings.ReplaceAll(cleanSel, "AWAY", "2")
	cleanSel = strings.ReplaceAll(cleanSel, "GG", "YES")
	cleanSel = strings.ReplaceAll(cleanSel, "NG", "NO")

	parts := strings.Split(cleanSel, "&")
	if len(parts) != 2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Invalid combo selection code '%s'", bet.SelectionCode), RequiredDomain: DomainScore}
	}
	selResult := strings.TrimSpace(parts[0])
	selBTTS := strings.TrimSpace(parts[1])
	won := selResult == resultOutcome && selBTTS == bttsOutcome

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Match Result and Over/Under (FB_RESULT_TOTALS)
// ----------------------------------------------------------------------------
type FootballResultTotalsRule struct{}

func (r *FootballResultTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_RESULT_TOTALS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballResultTotalsRule) RequiredDomain() string { return DomainScore }
func (r *FootballResultTotalsRule) Timing() string         { return TimingEventEnd }

func (r *FootballResultTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	if bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for Result + Totals", RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	resultOutcome := "X"
	if h > a {
		resultOutcome = "1"
	} else if a > h {
		resultOutcome = "2"
	}

	line := *bet.Line
	total := float64(h + a)

	if total == line {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Total goals equaled line (push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	totalsOutcome := "UNDER"
	if total > line {
		totalsOutcome = "OVER"
	}

	cleanSel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	cleanSel = strings.ReplaceAll(cleanSel, "AND", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "+", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "/", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "HOME", "1")
	cleanSel = strings.ReplaceAll(cleanSel, "DRAW", "X")
	cleanSel = strings.ReplaceAll(cleanSel, "AWAY", "2")

	parts := strings.Split(cleanSel, "&")
	if len(parts) != 2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Invalid combo selection code '%s'", bet.SelectionCode), RequiredDomain: DomainScore}
	}
	selResult := strings.TrimSpace(parts[0])
	selTotals := strings.TrimSpace(parts[1])
	won := selResult == resultOutcome && selTotals == totalsOutcome

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}

// ----------------------------------------------------------------------------
// Both Teams To Score and Over/Under (FB_BTTS_TOTALS)
// ----------------------------------------------------------------------------
type FootballBTTSTotalsRule struct{}

func (r *FootballBTTSTotalsRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_BTTS_TOTALS", Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballBTTSTotalsRule) RequiredDomain() string { return DomainScore }
func (r *FootballBTTSTotalsRule) Timing() string         { return TimingEventEnd }

func (r *FootballBTTSTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	if !state.IsFinished {
		return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingEventEnd, RequiredDomain: DomainScore}
	}
	if state.Score.Home < 0 || state.Score.Away < 0 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Negative score data detected", RequiredDomain: DomainScore}
	}
	if bet.Line == nil {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: "Missing required line specifier for BTTS + Totals", RequiredDomain: DomainScore}
	}

	h, a := state.Score.Home, state.Score.Away
	bttsOutcome := "NO"
	if h >= 1 && a >= 1 {
		bttsOutcome = "YES"
	}

	line := *bet.Line
	total := float64(h + a)

	if total == line {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Total goals equaled line (push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}

	totalsOutcome := "UNDER"
	if total > line {
		totalsOutcome = "OVER"
	}

	cleanSel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	cleanSel = strings.ReplaceAll(cleanSel, "AND", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "+", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "/", "&")
	cleanSel = strings.ReplaceAll(cleanSel, "GG", "YES")
	cleanSel = strings.ReplaceAll(cleanSel, "NG", "NO")

	parts := strings.Split(cleanSel, "&")
	if len(parts) != 2 {
		return SettlementResult{Status: BetStatusSettlementHold, Settled: false, Reason: fmt.Sprintf("Invalid combo selection code '%s'", bet.SelectionCode), RequiredDomain: DomainScore}
	}
	selBTTS := strings.TrimSpace(parts[0])
	selTotals := strings.TrimSpace(parts[1])
	won := selBTTS == bttsOutcome && selTotals == totalsOutcome

	if won {
		return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
}
