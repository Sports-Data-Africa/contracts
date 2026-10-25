package market_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type InventoryReportTest struct {
	TotalDefinitions                  int `json:"total_definitions"`
	MappedDefinitions                 int `json:"mapped_definitions"`
	CertifiedDefinitions              int `json:"certified_definitions"`
	MappedNotCertifiedDefinitions     int `json:"mapped_not_certified_definitions"`
	PendingContractExtensions         int `json:"pending_contract_extensions"`
	QuarantinedDefinitions            int `json:"quarantined_definitions"`
	TotalBlockedFromBettingSettlement int `json:"total_blocked_from_betting_settlement"`
	Definitions                       []struct {
		SportID             int    `json:"sport_id"`
		SportName           string `json:"sport_name"`
		RawMarketID         string `json:"raw_market_id"`
		RawMarketName       string `json:"raw_market_name"`
		CanonicalMarketCode string `json:"canonical_market_code"`
		InventoryStatus     string `json:"inventory_status"`     // MAPPED, PENDING, QUARANTINED
		CertificationStatus string `json:"certification_status"` // CERTIFIED, NOT_CERTIFIED, QUARANTINED
		PlacementGating     string `json:"placement_gating"`
		SettlementGating    string `json:"settlement_gating"`
	} `json:"definitions"`
}

func TestPhase4_4_AuthoritativeInventoryAssertions(t *testing.T) {
	// Locate PHASE_4_4_INVENTORY_RECONCILIATION.json
	paths := []string{
		"../../PHASE_4_4_INVENTORY_RECONCILIATION.json",
		"../PHASE_4_4_INVENTORY_RECONCILIATION.json",
		"PHASE_4_4_INVENTORY_RECONCILIATION.json",
	}

	var data []byte
	var err error
	for _, p := range paths {
		if abs, absErr := filepath.Abs(p); absErr == nil {
			if data, err = os.ReadFile(abs); err == nil {
				break
			}
		}
	}
	if err != nil {
		t.Fatalf("failed to read PHASE_4_4_INVENTORY_RECONCILIATION.json: %v", err)
	}

	var rep InventoryReportTest
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatalf("failed to unmarshal inventory report: %v", err)
	}

	// 1. Assert Total Definitions
	if rep.TotalDefinitions != 169 {
		t.Errorf("expected 169 total definitions, got %d", rep.TotalDefinitions)
	}
	if len(rep.Definitions) != 169 {
		t.Errorf("expected 169 definitions in array, got %d", len(rep.Definitions))
	}

	// 2. Assert Mapped Breakdown
	if rep.MappedDefinitions != 53 {
		t.Errorf("expected 53 mapped definitions, got %d", rep.MappedDefinitions)
	}
	if rep.CertifiedDefinitions != 46 {
		t.Errorf("expected 46 certified definitions, got %d", rep.CertifiedDefinitions)
	}
	if rep.MappedNotCertifiedDefinitions != 7 {
		t.Errorf("expected 7 mapped but uncertified definitions, got %d", rep.MappedNotCertifiedDefinitions)
	}
	if rep.CertifiedDefinitions+rep.MappedNotCertifiedDefinitions != rep.MappedDefinitions {
		t.Errorf("arithmetic mismatch: certified(%d) + mapped_not_certified(%d) != mapped(%d)",
			rep.CertifiedDefinitions, rep.MappedNotCertifiedDefinitions, rep.MappedDefinitions)
	}

	// 3. Assert Pending & Quarantined Breakdown
	if rep.PendingContractExtensions != 56 {
		t.Errorf("expected 56 pending definitions, got %d", rep.PendingContractExtensions)
	}
	if rep.QuarantinedDefinitions != 60 {
		t.Errorf("expected 60 quarantined definitions, got %d", rep.QuarantinedDefinitions)
	}

	// 4. Assert Blocked Totals (The 116 vs 123 reconciliation)
	if rep.TotalBlockedFromBettingSettlement != 123 {
		t.Errorf("expected 123 blocked definitions, got %d", rep.TotalBlockedFromBettingSettlement)
	}
	expectedBlocked := rep.MappedNotCertifiedDefinitions + rep.PendingContractExtensions + rep.QuarantinedDefinitions
	if rep.TotalBlockedFromBettingSettlement != expectedBlocked {
		t.Errorf("arithmetic mismatch: total blocked(%d) != mapped_not_cert(%d) + pending(%d) + quarantined(%d)",
			rep.TotalBlockedFromBettingSettlement, rep.MappedNotCertifiedDefinitions, rep.PendingContractExtensions, rep.QuarantinedDefinitions)
	}

	// 5. Assert 100% Exclusivity and Identity Uniqueness
	seenKeys := make(map[string]bool)
	var mappedCount, pendingCount, quarantinedCount, certCount int

	for _, d := range rep.Definitions {
		compositeKey := fmt.Sprintf("%d:%s", d.SportID, d.RawMarketID)
		if seenKeys[compositeKey] {
			t.Errorf("DUPLICATE PROVIDER DEFINITION DETECTED: key %s", compositeKey)
		}
		seenKeys[compositeKey] = true

		switch d.InventoryStatus {
		case "MAPPED":
			mappedCount++
			if d.CertificationStatus == "CERTIFIED" {
				certCount++
				if d.PlacementGating != "ALLOWED_SHADOW_ONLY" {
					t.Errorf("certified definition %s placement gating must be ALLOWED_SHADOW_ONLY, got %s",
						compositeKey, d.PlacementGating)
				}
			} else {
				if d.PlacementGating != "BLOCKED_FAIL_CLOSED" {
					t.Errorf("uncertified mapped definition %s must be BLOCKED_FAIL_CLOSED", compositeKey)
				}
			}
		case "PENDING":
			pendingCount++
			if d.PlacementGating != "BLOCKED_FAIL_CLOSED" {
				t.Errorf("pending definition %s must be BLOCKED_FAIL_CLOSED", compositeKey)
			}
		case "QUARANTINED":
			quarantinedCount++
			if d.PlacementGating != "BLOCKED_FAIL_CLOSED" {
				t.Errorf("quarantined definition %s must be BLOCKED_FAIL_CLOSED", compositeKey)
			}
		default:
			t.Errorf("unrecognized inventory status: %s for %s", d.InventoryStatus, compositeKey)
		}
	}

	if mappedCount != 53 || pendingCount != 56 || quarantinedCount != 60 || certCount != 46 {
		t.Errorf("counted mismatch: mapped=%d, pending=%d, quarantined=%d, cert=%d",
			mappedCount, pendingCount, quarantinedCount, certCount)
	}

	t.Logf("CONFIRMED: All 169 definitions accounted for with mathematical exclusivity: 46 Certified + 7 Mapped-Not-Cert + 56 Pending + 60 Quarantined = 169 Total (123 Blocked).")
}
