package market_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Sports-Data-Africa/contracts/market"
)

// 1. Deterministic mappings for captured provider payloads across sports
func TestPhase2_01_DeterministicMappingsAcrossSports(t *testing.T) {
	testCases := []struct {
		name          string
		input         market.ProviderMarketInput
		expectMapped  bool
		expectCode    string
		expectPeriod  string
		expectFormula string
		expectMetric  string
		expectNormSel string
	}{
		{
			name: "Football: 1X2 Full Time",
			input: market.ProviderMarketInput{
				Provider:       market.ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "1",
				RawMarketName:  "1X2",
				RawOutcomeName: "Home",
			},
			expectMapped:  true,
			expectCode:    "FB_1X2",
			expectPeriod:  market.PeriodFullTime,
			expectFormula: market.Formula1X2,
			expectMetric:  market.MetricGoals,
			expectNormSel: "1",
		},
		{
			name: "Football: 1st Half 1X2",
			input: market.ProviderMarketInput{
				Provider:       market.ProviderSportyBet,
				SportID:        1,
				RawMarketID:    "60",
				RawMarketName:  "1st Half - 1X2",
				RawOutcomeName: "Draw",
			},
			expectMapped:  true,
			expectCode:    "FB_HT_1X2",
			expectPeriod:  market.PeriodHalfTime,
			expectFormula: market.Formula1X2,
			expectMetric:  market.MetricGoals,
			expectNormSel: "X",
		},
		{
			name: "Basketball: Match Winner incl OT",
			input: market.ProviderMarketInput{
				Provider:       market.ProviderSportyBet,
				SportID:        2,
				RawMarketID:    "219",
				RawMarketName:  "Winner (incl. overtime)",
				RawOutcomeName: "Home",
			},
			expectMapped:  true,
			expectCode:    "BB_MONEYLINE",
			expectPeriod:  market.PeriodFullTime,
			expectFormula: market.FormulaMoneyline,
			expectMetric:  market.MetricGoals,
			expectNormSel: "1",
		},
		{
			name: "Tennis: Match Winner",
			input: market.ProviderMarketInput{
				Provider:       market.ProviderSportyBet,
				SportID:        3,
				RawMarketID:    "1",
				RawMarketName:  "Winner",
				RawOutcomeName: "Player 1",
			},
			expectMapped:  true,
			expectCode:    "TN_MATCH_WINNER",
			expectPeriod:  market.PeriodFullTime,
			expectFormula: market.FormulaMoneyline,
			expectMetric:  market.MetricGoals,
			expectNormSel: "PLAYER 1",
		},
		{
			name: "Ice Hockey: Regular Time 1X2",
			input: market.ProviderMarketInput{
				Provider:       market.ProviderSportyBet,
				SportID:        4,
				RawMarketID:    "1",
				RawMarketName:  "1X2",
				RawOutcomeName: "Away",
			},
			expectMapped:  true,
			expectCode:    "IH_REGULATION_1X2",
			expectPeriod:  market.PeriodFullTime,
			expectFormula: market.Formula1X2,
			expectMetric:  market.MetricGoals,
			expectNormSel: "2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := market.MapProviderMarket(tc.input)
			if res.Mapped != tc.expectMapped {
				t.Fatalf("Mapped status mismatch: got %v, expected %v (reason: %s)", res.Mapped, tc.expectMapped, res.Reason)
			}
			if res.CanonicalMarketCode != tc.expectCode {
				t.Errorf("Canonical code mismatch: got %s, expected %s", res.CanonicalMarketCode, tc.expectCode)
			}
			if res.Period != tc.expectPeriod {
				t.Errorf("Period mismatch: got %s, expected %s", res.Period, tc.expectPeriod)
			}
			if res.FormulaType != tc.expectFormula {
				t.Errorf("Formula mismatch: got %s, expected %s", res.FormulaType, tc.expectFormula)
			}
			if res.Metric != tc.expectMetric {
				t.Errorf("Metric mismatch: got %s, expected %s", res.Metric, tc.expectMetric)
			}
			if res.NormalizedSelection != tc.expectNormSel {
				t.Errorf("Selection normalization mismatch: got %s, expected %s", res.NormalizedSelection, tc.expectNormSel)
			}
		})
	}
}

// 2. Correct handling of specifiers and period distinctions
func TestPhase2_02_SpecifiersAndPeriodDistinctions(t *testing.T) {
	// Total Goals Over 2.5 vs Over 3.5 in full time
	input25 := market.ProviderMarketInput{
		Provider:       market.ProviderSportyBet,
		SportID:        1,
		RawMarketID:    "18",
		RawMarketName:  "Over/Under",
		RawSpecifier:   "total=2.5",
		RawOutcomeName: "Over 2.5",
	}
	res25 := market.MapProviderMarket(input25)
	if res25.Line == nil || *res25.Line != 2.5 {
		t.Fatalf("Expected line 2.5, got %v", res25.Line)
	}
	if res25.Period != market.PeriodFullTime {
		t.Errorf("Expected FT period, got %s", res25.Period)
	}

	// 1st Half Total Goals Over 1.5
	inputHT := market.ProviderMarketInput{
		Provider:       market.ProviderSportyBet,
		SportID:        1,
		RawMarketID:    "68",
		RawMarketName:  "1st Half - Over/Under",
		RawSpecifier:   "total=1.5",
		RawOutcomeName: "Over 1.5",
	}
	resHT := market.MapProviderMarket(inputHT)
	if resHT.Line == nil || *resHT.Line != 1.5 {
		t.Fatalf("Expected line 1.5, got %v", resHT.Line)
	}
	if resHT.Period != market.PeriodHalfTime {
		t.Errorf("Expected HT period, got %s", resHT.Period)
	}
}

// 3. Unknown, Player Prop, and Exotic Combo Markets Retention (Fail-Closed)
func TestPhase2_03_UnknownAndQuarantinedMarketsFailClosed(t *testing.T) {
	// Player Prop: 1st Goalscorer (ID 891)
	inputProp := market.ProviderMarketInput{
		Provider:       market.ProviderSportyBet,
		SportID:        1,
		RawMarketID:    "891",
		RawMarketName:  "1st Goalscorer & 1X2",
		RawSpecifier:   "goalnr=1|player=sr:player:1240036",
		RawOutcomeName: "Player A & Home",
	}
	resProp := market.MapProviderMarket(inputProp)
	if resProp.Status != market.MarketStatusDisabledUnsupported {
		t.Errorf("Player prop must be DISABLED_UNSUPPORTED in shadow mode, got: %s", resProp.Status)
	}

	// Early Payout Special: 1X2 - 2UP (ID 60100)
	inputSpecial := market.ProviderMarketInput{
		Provider:       market.ProviderSportyBet,
		SportID:        1,
		RawMarketID:    "60100",
		RawMarketName:  "1X2 - 2UP",
		RawOutcomeName: "Home",
	}
	resSpecial := market.MapProviderMarket(inputSpecial)
	if resSpecial.Status != market.MarketStatusDisabledUnsupported {
		t.Errorf("Early payout special must be DISABLED_UNSUPPORTED in shadow mode, got: %s", resSpecial.Status)
	}
}

// 4. Repeatable catalogue generation and reconciliation check
func TestPhase2_04_ReconciliationIntegrityFromJSON(t *testing.T) {
	matrixBytes, err := os.ReadFile("d:/Sports-Data-Africa/MARKET_MAPPING_MATRIX.json")
	if err != nil {
		t.Skip("Skipping reconciliation test: MARKET_MAPPING_MATRIX.json not found in root")
	}

	var data struct {
		TotalDefinitions int `json:"total_definitions"`
		Mappings          []struct {
			CanonicalSportID    int    `json:"canonical_sport_id"`
			NormalizationStatus string `json:"normalization_status"`
		} `json:"mappings"`
	}

	if err := json.Unmarshal(matrixBytes, &data); err != nil {
		t.Fatalf("Failed to parse MARKET_MAPPING_MATRIX.json: %v", err)
	}

	if data.TotalDefinitions != len(data.Mappings) {
		t.Fatalf("Header count %d != mapped entries %d", data.TotalDefinitions, len(data.Mappings))
	}

	counts := make(map[string]int)
	for _, m := range data.Mappings {
		counts[m.NormalizationStatus]++
	}

	sum := counts["MAPPED_CANONICAL"] + counts["UNMAPPED_PLAYER_PROP"] + counts["UNMAPPED_EXOTIC_COMBO"] +
		counts["UNMAPPED_EARLY_PAYOUT"] + counts["UNMAPPED_PENDING_CONTRACT"] + counts["AMBIGUOUS"]

	if sum != data.TotalDefinitions {
		t.Fatalf("Reconciliation sum %d != total definitions %d", sum, data.TotalDefinitions)
	}
}
