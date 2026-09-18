package canonical

import (
	"testing"
	"time"
)

func TestCanonicalFixtureIDFormatting(t *testing.T) {
	tests := []struct {
		sportID  int
		seq      int64
		expected string
	}{
		{SportIDFootball, 456778, "SPD-FB-456778"},
		{SportIDBasketball, 293841, "SPD-BB-293841"},
		{SportIDTennis, 582910, "SPD-TN-582910"},
		{SportIDCricket, 682910, "SPD-CK-682910"},
		{SportIDIceHockey, 12, "SPD-IH-000012"},
		{SportIDTableTennis, 999, "SPD-TT-000999"},
	}

	for _, tt := range tests {
		got, err := FormatCanonicalFixtureID(tt.sportID, tt.seq)
		if err != nil {
			t.Fatalf("unexpected error formatting sport %d, seq %d: %v", tt.sportID, tt.seq, err)
		}
		if got != tt.expected {
			t.Errorf("FormatCanonicalFixtureID(%d, %d) = %s, expected %s", tt.sportID, tt.seq, got, tt.expected)
		}
	}
}

func TestCanonicalFixtureIDParsing(t *testing.T) {
	tests := []struct {
		input       string
		expectedSid int
		expectedCod string
		expectedSeq int64
		shouldErr   bool
	}{
		{"SPD-FB-456778", SportIDFootball, "FB", 456778, false},
		{"SPD-BB-293841", SportIDBasketball, "BB", 293841, false},
		{"SPD-TN-000123", SportIDTennis, "TN", 123, false},
		{"SPD-CK-999999", SportIDCricket, "CK", 999999, false},
		{"INVALID-ID", 0, "", 0, true},
		{"SPD-XX-123456", 0, "", 0, true},
		{"SPD-FB-12345", 0, "", 0, true}, // 5 digits
		{"SPD-FB-1234567", 0, "", 0, true}, // 7 digits
	}

	for _, tt := range tests {
		sid, cod, seq, err := ParseCanonicalFixtureID(tt.input)
		if tt.shouldErr {
			if err == nil {
				t.Errorf("expected error for input %s, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
			if sid != tt.expectedSid || cod != tt.expectedCod || seq != tt.expectedSeq {
				t.Errorf("ParseCanonicalFixtureID(%s) = (%d, %s, %d), expected (%d, %s, %d)",
					tt.input, sid, cod, seq, tt.expectedSid, tt.expectedCod, tt.expectedSeq)
			}
		}
	}
}

func TestNaturalIdentityHash(t *testing.T) {
	t1 := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 9, 18, 20, 0, 0, 0, time.UTC)

	// Same team variations and identical time should yield identical hash
	h1 := NaturalIdentityHash(1, "Arsenal FC", "Chelsea FC", t1)
	h2 := NaturalIdentityHash(1, "arsenal", "chelsea", t2)
	if h1 != h2 {
		t.Errorf("expected hashes to match for normalized names: %s != %s", h1, h2)
	}

	// Different kickoff time should yield different hash
	h3 := NaturalIdentityHash(1, "Arsenal", "Chelsea", t3)
	if h1 == h3 {
		t.Errorf("different kickoff times must not collide: %s == %s", h1, h3)
	}
}
