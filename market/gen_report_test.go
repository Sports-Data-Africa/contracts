package market

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestGenerateCapabilityMatrixReport(t *testing.T) {
	matrix := GenerateCapabilityMatrix()

	var sb strings.Builder
	sb.WriteString("# Operator Capability Matrix Report\n\n")
	sb.WriteString("**Source of Truth**: Dynamically generated from Platform Canonical Registry (`contracts/market`)\n")
	sb.WriteString("**Execution Mode**: SHADOW (Automated Real-Money Financial Settlement strictly Disabled)\n")
	sb.WriteString(fmt.Sprintf("**Total Canonical Definitions Registered**: %d  \n", matrix.TotalCapabilities))
	sb.WriteString(fmt.Sprintf("**Certified Active Capabilities**: %d  \n", matrix.ActiveCount))
	sb.WriteString(fmt.Sprintf("**Suspended Capabilities**: %d  \n", matrix.SuspendedCount))
	sb.WriteString(fmt.Sprintf("**Quarantined Capabilities**: %d  \n", matrix.QuarantinedCount))
	sb.WriteString(fmt.Sprintf("**Disabled Capabilities**: %d  \n\n", matrix.DisabledCount))

	sb.WriteString("---\n\n")
	sb.WriteString("## 1. Summary by Sport\n\n")
	sb.WriteString("| Sport ID | Sport Name | Released | Active | Suspended | Quarantined | Disabled | Status Summary |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	// Group by sport
	sportStats := make(map[int]struct {
		Name        string
		Released    bool
		Active      int
		Suspended   int
		Quarantined int
		Disabled    int
	})

	for _, c := range matrix.Capabilities {
		s := sportStats[c.SportID]
		s.Name = c.SportName
		s.Released = c.SportReleased
		switch c.Status {
		case MarketStatusActive:
			s.Active++
		case MarketStatusSuspended:
			s.Suspended++
		case MarketStatusQuarantined:
			s.Quarantined++
		default:
			s.Disabled++
		}
		sportStats[c.SportID] = s
	}

	var sportIDs []int
	for id := range sportStats {
		sportIDs = append(sportIDs, id)
	}
	sort.Ints(sportIDs)

	for _, id := range sportIDs {
		s := sportStats[id]
		relStr := "No (Quarantined)"
		if s.Released {
			relStr = "Yes"
		}
		statusSum := "QUARANTINED"
		if s.Released {
			if s.Active > 0 && (s.Suspended > 0 || s.Disabled > 0) {
				statusSum = "PARTIAL_ACTIVE"
			} else if s.Active > 0 {
				statusSum = "FULLY_ACTIVE"
			} else {
				statusSum = "SUSPENDED"
			}
		}
		sb.WriteString(fmt.Sprintf("| **%d** | %s | %s | %d | %d | %d | %d | **%s** |\n",
			id, s.Name, relStr, s.Active, s.Suspended, s.Quarantined, s.Disabled, statusSum))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("## 2. Granular Market Capability Definitions\n\n")
	sb.WriteString("| Sport | Canonical Code | Market Name | Period | Required Domain | Domain Certified | Period Supported | Status | Reason |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	// Sort capabilities by SportID, then CanonicalMarketCode, then Period
	items := make([]CapabilityItem, len(matrix.Capabilities))
	copy(items, matrix.Capabilities)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SportID != items[j].SportID {
			return items[i].SportID < items[j].SportID
		}
		if items[i].CanonicalMarketCode != items[j].CanonicalMarketCode {
			return items[i].CanonicalMarketCode < items[j].CanonicalMarketCode
		}
		return items[i].Period < items[j].Period
	})

	for _, item := range items {
		domCertStr := "No"
		if item.DomainCertified {
			domCertStr = "Yes"
		}
		perSuppStr := "No"
		if item.PeriodSupported {
			perSuppStr = "Yes"
		}
		reasonStr := item.Reason
		if reasonStr == "" {
			reasonStr = "Certified & Validated"
		}
		sb.WriteString(fmt.Sprintf("| %s (%d) | `%s` | %s | `%s` | `%s` | %s | %s | **%s** | %s |\n",
			item.SportName, item.SportID, item.CanonicalMarketCode, item.MarketName, item.Period,
			item.RequiredResultDomain, domCertStr, perSuppStr, item.Status, reasonStr))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("## 3. Strict Operating Invariants\n\n")
	sb.WriteString("1. **Single Canonical Gate**: All bet placements are validated solely through `CanPlaceMarketCanonical`. No raw-string allowlists or legacy compatibility bypasses.\n")
	sb.WriteString("2. **No Fabricated / Dummy Lines**: Every line-dependent market (totals, spreads, handicaps) must supply a deterministic line extracted from provider telemetry. Requests missing lines are rejected immediately.\n")
	sb.WriteString("3. **Fail-Closed Settlement**: In the event of missing period evidence, ambiguous selection mapping, or uncertified domain data, tickets are quarantined to `REVIEW` with zero automated financial mutations.\n")
	sb.WriteString("4. **Operator Trading Controls**: The operations dashboard can suspend any active market at will, but can never enable an uncertified or quarantined market.\n")
	sb.WriteString("5. **Shadow Mode Isolation**: Real-money automatic payout remains strictly disabled (`SETTLEMENT_MODE=SHADOW`) until complete multi-service integration certification is performed.\n")

	outPath := "../../CAPABILITY_MATRIX_REPORT.md"
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("failed to write capability report: %v", err)
	}
	t.Logf("Successfully regenerated %s with %d definitions", outPath, matrix.TotalCapabilities)
}
