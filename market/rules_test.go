package market

import (
	"testing"
	"time"
)

// TestInstantIrreversibleSettlement verifies that mathematically locked bets
// resolve immediately while play continues, without waiting for FT.
func TestInstantIrreversibleSettlement(t *testing.T) {
	now := time.Now()

	// 1. Football Over 2.5 & Under 2.5 at 23' (Score 2-1, Match LIVE)
	live2_1State := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100001",
		SportID:            1,
		Status:             FixtureStatusLive,
		IsFinished:         false,
		Score:              ScoreData{Home: 2, Away: 1}, // 3 goals
		ScoreConfirmedAt:   now.Add(-3 * time.Minute),
	}

	line2_5 := 2.5
	over2_5Bet := BetSelection{
		BetID:               "bet_over_25",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              PeriodFullTime,
		SelectionCode:       "OVER",
		Line:                &line2_5,
		RuleVersion:         1,
	}
	under2_5Bet := BetSelection{
		BetID:               "bet_under_25",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              PeriodFullTime,
		SelectionCode:       "UNDER",
		Line:                &line2_5,
		RuleVersion:         1,
	}

	resOver, err := Evaluate(live2_1State, over2_5Bet)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if resOver.Status != BetStatusWon || !resOver.Settled || resOver.Timing != TimingInstantIrreversible {
		t.Errorf("Over 2.5 at 2-1 should be INSTANT WON, got status=%s, settled=%v, timing=%s", resOver.Status, resOver.Settled, resOver.Timing)
	}

	resUnder, err := Evaluate(live2_1State, under2_5Bet)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if resUnder.Status != BetStatusLost || !resUnder.Settled || resUnder.Timing != TimingInstantIrreversible {
		t.Errorf("Under 2.5 at 2-1 should be INSTANT LOST, got status=%s, settled=%v, timing=%s", resUnder.Status, resUnder.Settled, resUnder.Timing)
	}

	// 2. Football BTTS YES & NO at 35' (Score 1-1, Match LIVE)
	live1_1State := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100002",
		SportID:            1,
		Status:             FixtureStatusLive,
		IsFinished:         false,
		Score:              ScoreData{Home: 1, Away: 1},
		ScoreConfirmedAt:   now.Add(-3 * time.Minute),
	}

	bttsYesBet := BetSelection{
		BetID:               "bet_btts_yes",
		SportID:             1,
		CanonicalMarketCode: "FB_BTTS",
		Period:              PeriodFullTime,
		SelectionCode:       "YES",
		RuleVersion:         1,
	}
	bttsNoBet := BetSelection{
		BetID:               "bet_btts_no",
		SportID:             1,
		CanonicalMarketCode: "FB_BTTS",
		Period:              PeriodFullTime,
		SelectionCode:       "NO",
		RuleVersion:         1,
	}

	resBTTSYes, _ := Evaluate(live1_1State, bttsYesBet)
	if resBTTSYes.Status != BetStatusWon || !resBTTSYes.Settled {
		t.Errorf("BTTS YES at 1-1 should be INSTANT WON, got %s", resBTTSYes.Status)
	}

	resBTTSNo, _ := Evaluate(live1_1State, bttsNoBet)
	if resBTTSNo.Status != BetStatusLost || !resBTTSNo.Settled {
		t.Errorf("BTTS NO at 1-1 should be INSTANT LOST, got %s", resBTTSNo.Status)
	}

	// 3. Football HT Over 0.5 & HT Under 0.5 at 23' 1-0 (Match LIVE 1H)
	live1_0State := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100003",
		SportID:            1,
		Status:             FixtureStatusLive,
		IsFinished:         false,
		Score:              ScoreData{Home: 1, Away: 0},
		PeriodScores:       map[string]ScoreData{"1H": {Home: 1, Away: 0}},
		ScoreConfirmedAt:   now.Add(-3 * time.Minute),
	}

	line0_5 := 0.5
	htOver0_5Bet := BetSelection{
		BetID:               "bet_ht_over05",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              PeriodHalfTime,
		SelectionCode:       "OVER",
		Line:                &line0_5,
		RuleVersion:         1,
	}
	htUnder0_5Bet := BetSelection{
		BetID:               "bet_ht_under05",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              PeriodHalfTime,
		SelectionCode:       "UNDER",
		Line:                &line0_5,
		RuleVersion:         1,
	}

	resHTOver, _ := Evaluate(live1_0State, htOver0_5Bet)
	if resHTOver.Status != BetStatusWon || !resHTOver.Settled {
		t.Errorf("HT Over 0.5 at 1-0 should be INSTANT WON, got %s", resHTOver.Status)
	}

	resHTUnder, _ := Evaluate(live1_0State, htUnder0_5Bet)
	if resHTUnder.Status != BetStatusLost || !resHTUnder.Settled {
		t.Errorf("HT Under 0.5 at 1-0 should be INSTANT LOST, got %s", resHTUnder.Status)
	}

	// 4. Basketball Total Points Over 215.5 when score reaches 110 - 106 = 216 in Q4
	liveBBState := CanonicalEventState{
		CanonicalFixtureID: "SPD-BB-200001",
		SportID:            2,
		Status:             FixtureStatusLive,
		IsFinished:         false,
		Score:              ScoreData{Home: 110, Away: 106}, // 216 points
		ScoreConfirmedAt:   now.Add(-2 * time.Minute),
	}
	line215_5 := 215.5
	bbOverBet := BetSelection{
		BetID:               "bet_bb_over",
		SportID:             2,
		CanonicalMarketCode: "BB_TOTAL_POINTS",
		Period:              PeriodFullTime,
		SelectionCode:       "OVER",
		Line:                &line215_5,
		RuleVersion:         1,
	}
	resBBOver, _ := Evaluate(liveBBState, bbOverBet)
	if resBBOver.Status != BetStatusWon || !resBBOver.Settled {
		t.Errorf("Basketball Over 215.5 at 216 points should be INSTANT WON, got %s", resBBOver.Status)
	}
}

// TestPeriodSettlement verifies that period markets resolve at the end of their period,
// without waiting for full time match completion.
func TestPeriodSettlement(t *testing.T) {
	// Half Time 1X2 settled at HT (Score 1-0, Status HT)
	htState := CanonicalEventState{
		CanonicalFixtureID: "SPD-FB-100004",
		SportID:            1,
		Status:             FixtureStatusHT,
		IsFinished:         false, // Full time has NOT occurred!
		IsPeriodFinished:   map[string]bool{PeriodHalfTime: true},
		PeriodScores:       map[string]ScoreData{"1H": {Home: 1, Away: 0}},
	}

	htHomeBet := BetSelection{
		BetID:               "bet_ht_1",
		SportID:             1,
		CanonicalMarketCode: "FB_HT_1X2",
		Period:              PeriodHalfTime,
		SelectionCode:       "1",
		RuleVersion:         1,
	}
	htDrawBet := BetSelection{
		BetID:               "bet_ht_x",
		SportID:             1,
		CanonicalMarketCode: "FB_HT_1X2",
		Period:              PeriodHalfTime,
		SelectionCode:       "X",
		RuleVersion:         1,
	}

	resHTHome, err := Evaluate(htState, htHomeBet)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if resHTHome.Status != BetStatusWon || !resHTHome.Settled || resHTHome.Timing != TimingPeriodEnd {
		t.Errorf("HT 1 should be WON at halftime whistle, got %s (timing=%s)", resHTHome.Status, resHTHome.Timing)
	}

	resHTDraw, _ := Evaluate(htState, htDrawBet)
	if resHTDraw.Status != BetStatusLost || !resHTDraw.Settled {
		t.Errorf("HT X should be LOST at halftime whistle, got %s", resHTDraw.Status)
	}

	// But FT 1X2 on the SAME match MUST remain OPEN!
	ft1X2Bet := BetSelection{
		BetID:               "bet_ft_1",
		SportID:             1,
		CanonicalMarketCode: "FB_1X2",
		Period:              PeriodFullTime,
		SelectionCode:       "1",
		RuleVersion:         1,
	}
	resFT1X2, _ := Evaluate(htState, ft1X2Bet)
	if resFT1X2.Status != BetStatusOpen || resFT1X2.Settled {
		t.Errorf("FT 1X2 at halftime MUST remain OPEN, got status=%s, settled=%v", resFT1X2.Status, resFT1X2.Settled)
	}

	// Tennis Set 1 Winner resolves when Set 1 is finished
	tennisS1State := CanonicalEventState{
		CanonicalFixtureID: "SPD-TN-300001",
		SportID:            3,
		Status:             FixtureStatusLive,
		IsFinished:         false,
		TennisSets: []TennisSetData{
			{SetNumber: 1, HomeGames: 6, AwayGames: 4, IsFinished: true},
			{SetNumber: 2, HomeGames: 2, AwayGames: 1, IsFinished: false},
		},
	}
	tennisS1Bet := BetSelection{
		BetID:               "bet_tn_s1",
		SportID:             3,
		CanonicalMarketCode: "TN_SET_WINNER",
		Period:              PeriodSet1,
		SelectionCode:       "1",
		RuleVersion:         1,
	}
	resS1, _ := Evaluate(tennisS1State, tennisS1Bet)
	if resS1.Status != BetStatusWon || !resS1.Settled {
		t.Errorf("Tennis Set 1 Winner should resolve as WON while match is live, got %s", resS1.Status)
	}
}

// TestZeroUnsupportedMarketsInvariant verifies that unmapped markets or markets without
// canonical rules are immediately marked DISABLED_UNSUPPORTED and cannot accept bets.
func TestZeroUnsupportedMarketsInvariant(t *testing.T) {
	// Unknown market type from SportyBet
	res := MapProviderMarket(ProviderMarketInput{
		Provider:      ProviderSportyBet,
		SportID:       1,
		RawMarketID:   "99999",
		RawMarketName: "Player To Score From Own Half In Minute 42",
		RawMarketType: "EXOTIC_UNKNOWN",
	})

	if res.Status != MarketStatusDisabledUnsupported {
		t.Errorf("Unmapped market must have status DISABLED_UNSUPPORTED, got %s", res.Status)
	}
	if res.Mapped {
		t.Errorf("Unmapped market Mapped must be false")
	}

	// Legitimate market maps successfully
	validRes := MapProviderMarket(ProviderMarketInput{
		Provider:       ProviderSportyBet,
		SportID:        1,
		RawMarketID:    "1",
		RawMarketName:  "1X2",
		RawMarketType:  "1X2",
		RawOutcomeID:   "1",
		RawOutcomeName: "1",
	})

	if validRes.Status != MarketStatusActive || !validRes.Mapped || !validRes.HasRule {
		t.Errorf("Valid 1X2 market must be ACTIVE, mapped, and have rule, got %+v", validRes)
	}
	if validRes.CanonicalMarketCode != "FB_1X2" {
		t.Errorf("Expected canonical code FB_1X2, got %s", validRes.CanonicalMarketCode)
	}

	// Odibets GG maps to FB_BTTS
	odiRes := MapProviderMarket(ProviderMarketInput{
		Provider:       ProviderOdibets,
		SportID:        1,
		RawMarketID:    "odi_gg",
		RawMarketName:  "Both Teams To Score",
		RawMarketType:  "GG",
		RawOutcomeName: "Yes",
	})
	if odiRes.CanonicalMarketCode != "FB_BTTS" || odiRes.Status != MarketStatusActive {
		t.Errorf("Odibets GG must map to active FB_BTTS, got %+v", odiRes)
	}
}

// TestAllActiveSportsCoverage verifies that all 12 active sports have registered rules.
func TestAllActiveSportsCoverage(t *testing.T) {
	activeSports := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13}

	for _, sID := range activeSports {
		rulesFound := 0
		allKeys := ListAll()
		for _, k := range allKeys {
			if k.SportID == sID {
				rulesFound++
			}
		}
		if rulesFound == 0 {
			t.Errorf("Sport ID %d has ZERO registered settlement rules!", sID)
		}
	}
}
