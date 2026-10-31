package market

import (
	"testing"
)

func TestCricketRulesAndMapping(t *testing.T) {
	// 1. Test SportyBet mapping for Cricket 1X2, Draw No Bet, Double Chance
	res1X2 := MapProviderMarket(ProviderMarketInput{
		Provider:      "SPORTYBET",
		SportID:       8,
		RawMarketName: "1X2",
		RawMarketID:   "1",
		RawOutcomeID:  "1",
		RawOutcomeName: "Home",
	})
	if res1X2.CanonicalMarketCode != "CK_1X2" {
		t.Fatalf("expected CK_1X2, got %s (status: %s, reason: %s)", res1X2.CanonicalMarketCode, res1X2.Status, res1X2.Reason)
	}
	if res1X2.Status != MarketStatusActive {
		t.Fatalf("expected CK_1X2 to be active, got %s", res1X2.Status)
	}

	resDNB := MapProviderMarket(ProviderMarketInput{
		Provider:      "SPORTYBET",
		SportID:       8,
		RawMarketName: "Draw No Bet",
		RawMarketID:   "2",
		RawOutcomeID:  "1",
		RawOutcomeName: "Home",
	})
	if resDNB.CanonicalMarketCode != "CK_DNB" {
		t.Fatalf("expected CK_DNB, got %s", resDNB.CanonicalMarketCode)
	}
	if resDNB.Status != MarketStatusActive {
		t.Fatalf("expected CK_DNB to be active, got %s", resDNB.Status)
	}

	resDC := MapProviderMarket(ProviderMarketInput{
		Provider:      "SPORTYBET",
		SportID:       8,
		RawMarketName: "Double Chance",
		RawMarketID:   "3",
		RawOutcomeID:  "1",
		RawOutcomeName: "Home or Draw",
	})
	if resDC.CanonicalMarketCode != "CK_DOUBLE_CHANCE" {
		t.Fatalf("expected CK_DOUBLE_CHANCE, got %s", resDC.CanonicalMarketCode)
	}
	if resDC.Status != MarketStatusActive {
		t.Fatalf("expected CK_DOUBLE_CHANCE to be active, got %s", resDC.Status)
	}

	// 2. Test Rule Evaluation: 1X2
	rule1X2 := &Cricket1X2Rule{}
	stateDraw := CanonicalEventState{
		IsFinished: true,
		Cricket: &CricketStateData{
			MatchCompleted: true,
			Winner:         "DRAW",
		},
	}
	betX := BetSelection{SelectionCode: "X"}
	settleX := rule1X2.Evaluate(stateDraw, betX)
	if settleX.Status != BetStatusWon {
		t.Fatalf("expected BetStatusWon for X on DRAW, got %s", settleX.Status)
	}

	bet1 := BetSelection{SelectionCode: "1"}
	settle1 := rule1X2.Evaluate(stateDraw, bet1)
	if settle1.Status != BetStatusLost {
		t.Fatalf("expected BetStatusLost for 1 on DRAW, got %s", settle1.Status)
	}

	// 3. Test Rule Evaluation: DNB
	ruleDNB := &CricketDrawNoBetRule{}
	settleDNB := ruleDNB.Evaluate(stateDraw, bet1)
	if settleDNB.Status != BetStatusPush || settleDNB.PayoutFactor != 1.0 {
		t.Fatalf("expected BetStatusPush with 1.0 payout for DNB on DRAW, got %s (factor: %f)", settleDNB.Status, settleDNB.PayoutFactor)
	}

	// 4. Test Rule Evaluation: Double Chance
	ruleDC := &CricketDoubleChanceRule{}
	bet1X := BetSelection{SelectionCode: "1X"}
	settle1X := ruleDC.Evaluate(stateDraw, bet1X)
	if settle1X.Status != BetStatusWon {
		t.Fatalf("expected BetStatusWon for 1X on DRAW, got %s", settle1X.Status)
	}
}
