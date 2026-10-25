package canonical_test

import (
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

func TestProviderRegistry_NormalizationAndDisplay(t *testing.T) {
	tests := []struct {
		rawInput     string
		expectedCode string
		expectedDisp string
		expectedID   int
	}{
		// SportsXBet (formerly SportyBet) - Canonical and single exact legacy alias ONLY
		{"SPORTSXBET", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},
		{"SportsXBet", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},
		{"sportsxbet", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},
		{"SPORTYBET", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},
		{"SportyBet", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},
		{"sportybet", canonical.ProviderSportsXBet, canonical.DisplayNameSportsXBet, canonical.ProviderIDSportsXBet},

		// OddixBets (formerly Odibets) - Canonical and single exact legacy alias ONLY
		{"ODDIXBETS", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
		{"OddixBets", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
		{"oddixbets", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
		{"ODIBETS", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
		{"Odibets", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
		{"odibets", canonical.ProviderOddixBets, canonical.DisplayNameOddixBets, canonical.ProviderIDOddixBets},
	}

	for _, tt := range tests {
		t.Run(tt.rawInput, func(t *testing.T) {
			norm := canonical.NormalizeProvider(tt.rawInput)
			if norm != tt.expectedCode {
				t.Errorf("NormalizeProvider(%q) = %q, expected %q", tt.rawInput, norm, tt.expectedCode)
			}

			disp := canonical.GetProviderDisplayName(tt.rawInput)
			if disp != tt.expectedDisp {
				t.Errorf("GetProviderDisplayName(%q) = %q, expected %q", tt.rawInput, disp, tt.expectedDisp)
			}

			id := canonical.GetInternalProviderID(tt.rawInput)
			if id != tt.expectedID {
				t.Errorf("GetInternalProviderID(%q) = %d, expected %d", tt.rawInput, id, tt.expectedID)
			}
		})
	}

	// Verify that heuristic guesswork is NOT tolerated
	guessworkInputs := []string{"ODI", "SPORTY", "ODDIX", "SPORTSX", "SR", "PARIPESA", "UNKNOWN"}
	for _, gw := range guessworkInputs {
		norm := canonical.NormalizeProvider(gw)
		if norm == canonical.ProviderSportsXBet || norm == canonical.ProviderOddixBets {
			t.Errorf("GUESSWORK LEAK: input %q incorrectly guessed as canonical %q", gw, norm)
		}
	}
}

func TestProviderProvenance_AuditTraceability(t *testing.T) {
	now := time.Now().UTC()
	prov := canonical.CreateProvenance(
		"sportybet",        // Raw legacy name from wire
		"sportybet",        // Upstream protocol identifier
		"sr:match:987654",  // Raw event ID
		"18",               // Raw market ID
		"total_over",       // Raw outcome ID
		"total=2.5",        // Raw specifier
		now,
		1,                  // Mapping version
		"corr_trace_12345", // Correlation ID
	)

	// Invariants:
	// 1. Canonical provider code is standardized to SPORTSXBET
	if prov.CanonicalProviderName != canonical.ProviderSportsXBet {
		t.Errorf("expected CanonicalProviderName %s, got %s", canonical.ProviderSportsXBet, prov.CanonicalProviderName)
	}

	// 2. Display name is "SportsXBet"
	if prov.DisplayName != canonical.DisplayNameSportsXBet {
		t.Errorf("expected DisplayName %s, got %s", canonical.DisplayNameSportsXBet, prov.DisplayName)
	}

	// 3. Stable internal provider ID is 1
	if prov.InternalProviderID != 1 {
		t.Errorf("expected InternalProviderID 1, got %d", prov.InternalProviderID)
	}

	// 4. Upstream identifier is preserved intact for forensic auditing
	if prov.UpstreamProviderID != "sportybet" {
		t.Errorf("expected UpstreamProviderID 'sportybet', got %s", prov.UpstreamProviderID)
	}

	// 5. Raw IDs are preserved
	if prov.RawEventID != "sr:match:987654" || prov.RawMarketID != "18" || prov.RawSpecifiers != "total=2.5" {
		t.Errorf("raw IDs corrupted: %+v", prov)
	}
}

func TestProviderRegistry_ActiveValidation(t *testing.T) {
	if err := canonical.ValidateProviderActive("SportsXBet"); err != nil {
		t.Errorf("expected SportsXBet active, got: %v", err)
	}
	if err := canonical.ValidateProviderActive("sportybet"); err != nil {
		t.Errorf("expected legacy sportybet to resolve to active, got: %v", err)
	}
	if err := canonical.ValidateProviderActive("OddixBets"); err != nil {
		t.Errorf("expected OddixBets active, got: %v", err)
	}
	if err := canonical.ValidateProviderActive("odibets"); err != nil {
		t.Errorf("expected legacy odibets to resolve to active, got: %v", err)
	}
	if err := canonical.ValidateProviderActive("UNKNOWN_VENDOR"); err == nil {
		t.Errorf("expected unknown provider to fail validation, got nil")
	}
}
