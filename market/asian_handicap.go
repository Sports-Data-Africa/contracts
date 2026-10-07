package market

import (
	"fmt"
	"math"
)

// AsianOutcome represents the 5 canonical Asian handicap settlement results.
type AsianOutcome string

const (
	AsianWin      AsianOutcome = "WIN"
	AsianHalfWin  AsianOutcome = "HALF_WIN"
	AsianPush     AsianOutcome = "PUSH"
	AsianHalfLoss AsianOutcome = "HALF_LOSS"
	AsianLoss     AsianOutcome = "LOSS"
)

// FloatToLineQuarters converts a decimal line to an exact integer quarter representation.
// 0.00 -> 0
// 0.25 -> 1
// 0.50 -> 2
// 0.75 -> 3
// 1.00 -> 4
// -0.25 -> -1
// -0.50 -> -2
// -0.75 -> -3
// -1.00 -> -4
func FloatToLineQuarters(val float64) int {
	return int(math.Round(val * 4.0))
}

// LineQuartersToFloat converts an integer quarter representation back to a float for display.
func LineQuartersToFloat(quarters int) float64 {
	return float64(quarters) / 4.0
}

// EvaluateAsianHandicap evaluates an Asian Handicap bet using exact integer quarter arithmetic.
// No floating-point operations are performed in the settlement calculation.
//
// homeScore: final confirmed home score
// awayScore: final confirmed away score
// selection: "1" (Home) or "2" (Away)
// lineQuarters: the handicap line in integer quarters for the selected team
func EvaluateAsianHandicap(homeScore, awayScore int, selection string, lineQuarters int) (AsianOutcome, float64) {
	// If it's a quarter line (odd quarters: +/-1, +/-3, +/-5...), split into adjacent half-lines
	if lineQuarters%2 != 0 {
		lineA := lineQuarters - 1 // adjacent half line
		lineB := lineQuarters + 1 // adjacent half line

		resA, _ := evaluateSingleAsianHandicap(homeScore, awayScore, selection, lineA)
		resB, _ := evaluateSingleAsianHandicap(homeScore, awayScore, selection, lineB)

		return combineAsianSplit(resA, resB)
	}

	return evaluateSingleAsianHandicap(homeScore, awayScore, selection, lineQuarters)
}

func evaluateSingleAsianHandicap(homeScore, awayScore int, selection string, lineQuarters int) (AsianOutcome, float64) {
	var margin int
	switch selection {
	case "1", "HOME":
		// Home margin: (Home - Away) * 4 + lineQuarters
		margin = (homeScore-awayScore)*4 + lineQuarters
	case "2", "AWAY":
		// Away margin: (Away - Home) * 4 + lineQuarters
		margin = (awayScore-homeScore)*4 + lineQuarters
	default:
		return AsianLoss, 0.0
	}

	if margin > 0 {
		return AsianWin, 1.0
	} else if margin == 0 {
		return AsianPush, 1.0 // full refund of stake
	} else {
		return AsianLoss, 0.0
	}
}

// EvaluateAsianTotal evaluates an Asian Goal/Points Total using exact integer quarter arithmetic.
//
// totalScore: final confirmed total score (home + away)
// selection: "OVER" or "UNDER"
// lineQuarters: the total line in integer quarters (e.g. 2.25 -> 9, 2.50 -> 10, 2.75 -> 11)
func EvaluateAsianTotal(totalScore int, selection string, lineQuarters int) (AsianOutcome, float64) {
	if lineQuarters%2 != 0 {
		lineA := lineQuarters - 1
		lineB := lineQuarters + 1

		resA, _ := evaluateSingleAsianTotal(totalScore, selection, lineA)
		resB, _ := evaluateSingleAsianTotal(totalScore, selection, lineB)

		return combineAsianSplit(resA, resB)
	}

	return evaluateSingleAsianTotal(totalScore, selection, lineQuarters)
}

func evaluateSingleAsianTotal(totalScore int, selection string, lineQuarters int) (AsianOutcome, float64) {
	totalQuarters := totalScore * 4
	var margin int
	switch selection {
	case "OVER":
		margin = totalQuarters - lineQuarters
	case "UNDER":
		margin = lineQuarters - totalQuarters
	default:
		return AsianLoss, 0.0
	}

	if margin > 0 {
		return AsianWin, 1.0
	} else if margin == 0 {
		return AsianPush, 1.0
	} else {
		return AsianLoss, 0.0
	}
}

// combineAsianSplit combines the results of two split half-lines (50% stake each).
func combineAsianSplit(resA, resB AsianOutcome) (AsianOutcome, float64) {
	if resA == AsianWin && resB == AsianWin {
		return AsianWin, 1.0
	}
	if resA == AsianLoss && resB == AsianLoss {
		return AsianLoss, 0.0
	}
	if resA == AsianPush && resB == AsianPush {
		return AsianPush, 1.0
	}

	// One win, one push -> HALF_WIN (half won at full odds, half pushed/refunded)
	if (resA == AsianWin && resB == AsianPush) || (resA == AsianPush && resB == AsianWin) {
		return AsianHalfWin, 0.5
	}

	// One push, one loss -> HALF_LOSS (half refunded, half lost)
	if (resA == AsianPush && resB == AsianLoss) || (resA == AsianLoss && resB == AsianPush) {
		return AsianHalfLoss, 0.5
	}

	// Should not occur for adjacent half-lines
	return AsianPush, 1.0
}

// FormatAsianLine formats integer quarters to standard betting line string.
func FormatAsianLine(quarters int) string {
	val := float64(quarters) / 4.0
	if val >= 0 {
		return fmt.Sprintf("+%.2f", val)
	}
	return fmt.Sprintf("%.2f", val)
}
