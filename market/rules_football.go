package market

import (
	"fmt"
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
	Register(&FootballTeamTotalsRule{Team: "HOME"})
	Register(&FootballTeamTotalsRule{Team: "AWAY"})
	Register(&FootballCorners1X2Rule{})
	Register(&FootballCornersTotalRule{})
	Register(&FootballCardsTotalRule{})
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
			currentGoals = state.Score.Home + state.Score.Away
		}
		periodDone = state.IsPeriodFinished[PeriodHalfTime] || state.Status == FixtureStatusHT || state.IsFinished
	case PeriodSecondHalf:
		if p2, ok := state.PeriodScores["2H"]; ok {
			currentGoals = p2.Home + p2.Away
		} else {
			htGoals := 0
			if ht, ok := state.PeriodScores["1H"]; ok {
				htGoals = ht.Home + ht.Away
			}
			currentGoals = (state.Score.Home + state.Score.Away) - htGoals
		}
		periodDone = state.IsFinished
	default:
		currentGoals = state.Score.Home + state.Score.Away
		periodDone = state.IsFinished
	}

	line := 2.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

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
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	bothScored := h >= 1 && a >= 1

	// Instant Irreversible resolution
	if bothScored {
		if sel == "YES" || sel == "GG" {
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
		if sel == "NO" || sel == "NG" {
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
		if sel == "NO" || sel == "NG" {
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
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

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
	h, a := state.Score.Home, state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	if h == a {
		return SettlementResult{Status: BetStatusPush, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, Reason: "Match ended in a draw (DNB push)", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
	}
	if (sel == "1" || sel == "HOME") && h > a || (sel == "2" || sel == "AWAY") && a > h {
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
	handicap := 0
	if bet.Line != nil {
		handicap = int(*bet.Line)
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
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

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	h, a := htScore.Home, htScore.Away

	won := (sel == "1" || sel == "HOME") && h > a ||
		(sel == "X" || sel == "DRAW") && h == a ||
		(sel == "2" || sel == "AWAY") && a > h

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
	h2, ok := state.PeriodScores["2H"]
	if !ok {
		ht, okHT := state.PeriodScores["1H"]
		if !okHT {
			ht, okHT = state.PeriodScores["HT"]
		}
		if okHT {
			h2 = ScoreData{Home: state.Score.Home - ht.Home, Away: state.Score.Away - ht.Away}
		} else {
			return SettlementResult{Status: BetStatusManualReview, Settled: false, Reason: "Missing 1H score to derive 2H result", RequiredDomain: DomainPeriodScore}
		}
	}

	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	won := (sel == "1" || sel == "HOME") && h2.Home > h2.Away ||
		(sel == "X" || sel == "DRAW") && h2.Home == h2.Away ||
		(sel == "2" || sel == "AWAY") && h2.Away > h2.Home

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
	ht, ok := state.PeriodScores["1H"]
	if !ok {
		ht, ok = state.PeriodScores["HT"]
	}
	if !ok {
		return SettlementResult{Status: BetStatusManualReview, Settled: false, Reason: "Missing HT score for HT/FT settlement", RequiredDomain: DomainPeriodScore}
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
	total := state.Score.Home + state.Score.Away
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

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
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHomeCleanSheet := strings.Contains(sel, "HOME") || sel == "1" || sel == "YES"

	if isHomeCleanSheet {
		// Home clean sheet means Away scores 0. If Away scores >= 1, clean sheet is instantly LOST!
		if state.Score.Away >= 1 {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: "Away team scored, clean sheet lost", RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
		if state.IsFinished {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	} else {
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
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))
	isHome := sel == "1" || strings.Contains(sel, "HOME")

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
	} else {
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
	Team string // HOME or AWAY
}

func (r *FootballTeamTotalsRule) Key() MarketRuleKey {
	code := "FB_HOME_TOTALS"
	if r.Team == "AWAY" {
		code = "FB_AWAY_TOTALS"
	}
	return MarketRuleKey{SportID: 1, MarketCode: code, Period: PeriodFullTime, RuleVersion: 1}
}
func (r *FootballTeamTotalsRule) RequiredDomain() string { return DomainScore }
func (r *FootballTeamTotalsRule) Timing() string         { return TimingInstantIrreversible }

func (r *FootballTeamTotalsRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	goals := state.Score.Home
	if r.Team == "AWAY" {
		goals = state.Score.Away
	}
	line := 1.5
	if bet.Line != nil {
		line = *bet.Line
	}
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	switch sel {
	case "OVER":
		if float64(goals) > line {
			return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Team goals (%d) exceeded line (%.1f)", goals, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	case "UNDER":
		if float64(goals) > line {
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingInstantIrreversible, Reason: fmt.Sprintf("Team goals (%d) exceeded under line (%.1f)", goals, line), RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	if state.IsFinished {
		switch sel {
		case "OVER":
			if float64(goals) > line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		case "UNDER":
			if float64(goals) < line {
				return SettlementResult{Status: BetStatusWon, PayoutFactor: 1.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
			}
			return SettlementResult{Status: BetStatusLost, PayoutFactor: 0.0, Settled: true, Timing: TimingEventEnd, RequiredDomain: DomainScore, EvaluatedAt: time.Now()}
		}
	}

	return SettlementResult{Status: BetStatusOpen, Settled: false, Timing: TimingInstantIrreversible, RequiredDomain: DomainScore}
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
