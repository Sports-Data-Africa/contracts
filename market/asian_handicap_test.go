package market

import (
	"testing"
)

func TestAsianHandicapQuarterLines(t *testing.T) {
	tests := []struct {
		name          string
		homeScore     int
		awayScore     int
		selection     string
		lineQuarters  int
		wantOutcome   AsianOutcome
		wantPayoutFac float64
	}{
		// Line -0.25 (Quarters = -1): Splits into Line 0 and Line -0.5
		{
			name:          "-0.25 Home Win by 1 (1-0): Both Win",
			homeScore:     1,
			awayScore:     0,
			selection:     "HOME",
			lineQuarters:  -1,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},
		{
			name:          "-0.25 Draw (1-1): Line 0 Pushes, Line -0.5 Loses -> HALF_LOSS",
			homeScore:     1,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  -1,
			wantOutcome:   AsianHalfLoss,
			wantPayoutFac: 0.5,
		},
		{
			name:          "-0.25 Away Win (0-1): Both Lose -> LOSS",
			homeScore:     0,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  -1,
			wantOutcome:   AsianLoss,
			wantPayoutFac: 0.0,
		},

		// Line +0.25 (Quarters = +1): Splits into Line 0 and Line +0.5
		{
			name:          "+0.25 Draw (1-1): Line 0 Pushes, Line +0.5 Wins -> HALF_WIN",
			homeScore:     1,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  1,
			wantOutcome:   AsianHalfWin,
			wantPayoutFac: 0.5,
		},
		{
			name:          "+0.25 Home Win (2-1): Both Win -> WIN",
			homeScore:     2,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  1,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},
		{
			name:          "+0.25 Away Win (0-1): Both Lose -> LOSS",
			homeScore:     0,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  1,
			wantOutcome:   AsianLoss,
			wantPayoutFac: 0.0,
		},

		// Line -0.75 (Quarters = -3): Splits into Line -0.5 and Line -1.0
		{
			name:          "-0.75 Home Win by 1 (2-1): Line -0.5 Wins, Line -1.0 Pushes -> HALF_WIN",
			homeScore:     2,
			awayScore:     1,
			selection:     "HOME",
			lineQuarters:  -3,
			wantOutcome:   AsianHalfWin,
			wantPayoutFac: 0.5,
		},
		{
			name:          "-0.75 Home Win by 2 (2-0): Both Win -> WIN",
			homeScore:     2,
			awayScore:     0,
			selection:     "HOME",
			lineQuarters:  -3,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},
		{
			name:          "-0.75 Draw (0-0): Both Lose -> LOSS",
			homeScore:     0,
			awayScore:     0,
			selection:     "HOME",
			lineQuarters:  -3,
			wantOutcome:   AsianLoss,
			wantPayoutFac: 0.0,
		},

		// Line +0.75 (Quarters = +3): Splits into Line +0.5 and Line +1.0
		{
			name:          "+0.75 Away Lose by 1 (1-0): Line +0.5 Loses, Line +1.0 Pushes -> HALF_LOSS",
			homeScore:     1,
			awayScore:     0,
			selection:     "AWAY",
			lineQuarters:  3,
			wantOutcome:   AsianHalfLoss,
			wantPayoutFac: 0.5,
		},
		{
			name:          "+0.75 Draw (1-1): Both Win -> WIN",
			homeScore:     1,
			awayScore:     1,
			selection:     "AWAY",
			lineQuarters:  3,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},

		// Line 0.0 (Quarters = 0): DNB / Level
		{
			name:          "0.0 Draw (0-0): PUSH",
			homeScore:     0,
			awayScore:     0,
			selection:     "HOME",
			lineQuarters:  0,
			wantOutcome:   AsianPush,
			wantPayoutFac: 1.0,
		},
		{
			name:          "0.0 Home Win (1-0): WIN",
			homeScore:     1,
			awayScore:     0,
			selection:     "HOME",
			lineQuarters:  0,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutcome, gotFac := EvaluateAsianHandicap(tt.homeScore, tt.awayScore, tt.selection, tt.lineQuarters)
			if gotOutcome != tt.wantOutcome {
				t.Errorf("EvaluateAsianHandicap outcome = %v, want %v", gotOutcome, tt.wantOutcome)
			}
			if gotFac != tt.wantPayoutFac {
				t.Errorf("EvaluateAsianHandicap payout factor = %v, want %v", gotFac, tt.wantPayoutFac)
			}
		})
	}
}

func TestAsianTotalsQuarterLines(t *testing.T) {
	// Over/Under 2.25 (Quarters = 9): Splits into 2.0 (8) and 2.5 (10)
	tests := []struct {
		name          string
		totalScore    int
		selection     string
		lineQuarters  int
		wantOutcome   AsianOutcome
		wantPayoutFac float64
	}{
		{
			name:          "Over 2.25 with 2 goals: 2.0 pushes, 2.5 loses -> HALF_LOSS",
			totalScore:    2,
			selection:     "OVER",
			lineQuarters:  9, // 2.25
			wantOutcome:   AsianHalfLoss,
			wantPayoutFac: 0.5,
		},
		{
			name:          "Under 2.25 with 2 goals: 2.0 pushes, 2.5 wins -> HALF_WIN",
			totalScore:    2,
			selection:     "UNDER",
			lineQuarters:  9, // 2.25
			wantOutcome:   AsianHalfWin,
			wantPayoutFac: 0.5,
		},
		{
			name:          "Over 2.25 with 3 goals: Both win -> WIN",
			totalScore:    3,
			selection:     "OVER",
			lineQuarters:  9,
			wantOutcome:   AsianWin,
			wantPayoutFac: 1.0,
		},
		{
			name:          "Over 2.75 with 3 goals (11 quarters, split 2.5 & 3.0): 2.5 wins, 3.0 pushes -> HALF_WIN",
			totalScore:    3,
			selection:     "OVER",
			lineQuarters:  11, // 2.75
			wantOutcome:   AsianHalfWin,
			wantPayoutFac: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutcome, gotFac := EvaluateAsianTotal(tt.totalScore, tt.selection, tt.lineQuarters)
			if gotOutcome != tt.wantOutcome {
				t.Errorf("EvaluateAsianTotal outcome = %v, want %v", gotOutcome, tt.wantOutcome)
			}
			if gotFac != tt.wantPayoutFac {
				t.Errorf("EvaluateAsianTotal payout factor = %v, want %v", gotFac, tt.wantPayoutFac)
			}
		})
	}
}
