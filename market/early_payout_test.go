package market

import (
	"sync"
	"testing"
	"time"
)

func Test2UP_HomeTwoGoalLeadFollowedByDraw(t *testing.T) {
	// Scenario: Home establishes 2-0 lead at 35', final score finishes 2-2
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 12, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "evt_1"},
		{Sequence: 2, Minute: 35, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "evt_2"},
		{Sequence: 3, Minute: 60, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 1, EventID: "evt_3"},
		{Sequence: 4, Minute: 82, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 2, EventID: "evt_4"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_1",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 2},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, ok := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)
	if !ok {
		t.Fatalf("FB_1X2_2UP rule not registered")
	}

	// 1. Home Bet: MUST BE WON (Early Payout Satisfied)
	betHome := BetSelection{
		BetID:               "bet_home_1",
		CanonicalFixtureID:  state.CanonicalFixtureID,
		SportID:             1,
		CanonicalMarketCode: "FB_1X2_2UP",
		Period:              PeriodFullTime,
		SelectionCode:       "1",
		RuleVersion:         1,
	}
	resHome := rule.Evaluate(state, betHome)
	if resHome.Status != BetStatusWon {
		t.Errorf("Expected Home selection to be WON, got %s (reason: %s)", resHome.Status, resHome.Reason)
	}
	if resHome.PayoutFactor != 1.0 {
		t.Errorf("Expected Home PayoutFactor 1.0, got %f", resHome.PayoutFactor)
	}
	if resHome.Timing != TimingInstantIrreversible {
		t.Errorf("Expected TimingInstantIrreversible, got %s", resHome.Timing)
	}

	// 2. Draw Bet: MUST BE WON (FT Result is 2-2)
	betDraw := BetSelection{
		BetID:               "bet_draw_1",
		CanonicalFixtureID:  state.CanonicalFixtureID,
		SportID:             1,
		CanonicalMarketCode: "FB_1X2_2UP",
		Period:              PeriodFullTime,
		SelectionCode:       "X",
		RuleVersion:         1,
	}
	resDraw := rule.Evaluate(state, betDraw)
	if resDraw.Status != BetStatusWon {
		t.Errorf("Expected Draw selection to be WON at FT, got %s (reason: %s)", resDraw.Status, resDraw.Reason)
	}

	// 3. Away Bet: MUST BE LOST (Away never led by 2, and did not win FT)
	betAway := BetSelection{
		BetID:               "bet_away_1",
		CanonicalFixtureID:  state.CanonicalFixtureID,
		SportID:             1,
		CanonicalMarketCode: "FB_1X2_2UP",
		Period:              PeriodFullTime,
		SelectionCode:       "2",
		RuleVersion:         1,
	}
	resAway := rule.Evaluate(state, betAway)
	if resAway.Status != BetStatusLost {
		t.Errorf("Expected Away selection to be LOST, got %s (reason: %s)", resAway.Status, resAway.Reason)
	}
}

func Test2UP_HomeTwoGoalLeadFollowedByOpponentWin(t *testing.T) {
	// Scenario: Home establishes 2-0 lead at 28', but Away comes back to win 2-3 at FT
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 10, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "g1"},
		{Sequence: 2, Minute: 28, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "g2"},
		{Sequence: 3, Minute: 44, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 1, EventID: "g3"},
		{Sequence: 4, Minute: 67, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 2, EventID: "g4"},
		{Sequence: 5, Minute: 89, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 3, EventID: "g5"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_2",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 3},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Home: WON (2UP early trigger preserved)
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusWon {
		t.Errorf("Expected Home to be WON via early trigger, got %s", resHome.Status)
	}

	// Away: WON (Full-Time fallback win)
	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusWon {
		t.Errorf("Expected Away to be WON via FT fallback, got %s", resAway.Status)
	}

	// Draw: LOST (2-3)
	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusLost {
		t.Errorf("Expected Draw to be LOST, got %s", resDraw.Status)
	}
}

func Test2UP_AwayTwoGoalLeadFollowedByHomeWin(t *testing.T) {
	// Scenario: Away leads 0-2 at 40', Home comes back to win 3-2 at FT
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 15, ScoringTeam: "AWAY", HomeScoreAfter: 0, AwayScoreAfter: 1, EventID: "a1"},
		{Sequence: 2, Minute: 40, ScoringTeam: "AWAY", HomeScoreAfter: 0, AwayScoreAfter: 2, EventID: "a2"},
		{Sequence: 3, Minute: 55, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 2, EventID: "h1"},
		{Sequence: 4, Minute: 70, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 2, EventID: "h2"},
		{Sequence: 5, Minute: 88, ScoringTeam: "HOME", HomeScoreAfter: 3, AwayScoreAfter: 2, EventID: "h3"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_3",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 3, Away: 2},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Away: WON early payout
	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusWon {
		t.Errorf("Expected Away to be WON via 2UP trigger, got %s", resAway.Status)
	}

	// Home: WON via FT fallback
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusWon {
		t.Errorf("Expected Home to be WON via FT fallback, got %s", resHome.Status)
	}
}

func Test2UP_NoQualifyingLead_StandardFTFallback(t *testing.T) {
	// Scenario: 2-1 final score, but score progression was: 0-1, 1-1, 2-1. Neither team led by 2.
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 20, ScoringTeam: "AWAY", HomeScoreAfter: 0, AwayScoreAfter: 1, EventID: "t1"},
		{Sequence: 2, Minute: 50, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 1, EventID: "t2"},
		{Sequence: 3, Minute: 80, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 1, EventID: "t3"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_4",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 1},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Home: WON via FT fallback
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusWon {
		t.Errorf("Expected Home WON via FT fallback, got %s", resHome.Status)
	}

	// Draw: LOST
	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusLost {
		t.Errorf("Expected Draw LOST, got %s", resDraw.Status)
	}

	// Away: LOST
	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusLost {
		t.Errorf("Expected Away LOST, got %s", resAway.Status)
	}
}

func Test2UP_LeadAchievedLateInMatch(t *testing.T) {
	// Scenario: Home scores at 88' to make it 2-0. Match is still in-play (LIVE_2H).
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 25, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "late_1"},
		{Sequence: 2, Minute: 88, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "late_2"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_late",
		SportID:            1,
		Status:             "LIVE_2H",
		IsFinished:         false,
		Score:              ScoreData{Home: 2, Away: 0},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Home: Should immediately settle as WON (Instant Irreversible)
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusWon || !resHome.Settled {
		t.Errorf("Expected Home to settle WON immediately at 88', got status=%s settled=%v", resHome.Status, resHome.Settled)
	}

	// Draw & Away: Match still in play, so remain OPEN
	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusOpen {
		t.Errorf("Expected Draw OPEN while match in progress, got %s", resDraw.Status)
	}
}

func Test2UP_DuplicateAndOutOfOrderTimelineEvents(t *testing.T) {
	// Timeline received out of order and with duplicate event IDs:
	// Event 2 (Minute 40, Seq 2) arrives before Event 1 (Minute 15, Seq 1)
	// Event 2 is duplicated twice
	timeline := []GoalTimelineEntry{
		{Sequence: 2, Minute: 40, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "dup_event_2"},
		{Sequence: 1, Minute: 15, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "event_1"},
		{Sequence: 2, Minute: 40, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "dup_event_2"}, // Duplicate
	}

	home2Up, away2Up, homeMin, _, reliable := ReconstructTimeline(timeline)
	if !reliable {
		t.Errorf("Expected timeline to be reliable")
	}
	if !home2Up {
		t.Errorf("Expected home 2UP to be detected despite out-of-order delivery")
	}
	if away2Up {
		t.Errorf("Away should not be 2UP")
	}
	if homeMin != 40 {
		t.Errorf("Expected trigger minute 40, got %d", homeMin)
	}
}

func Test2UP_VARReversalOverturningQualifyingGoal(t *testing.T) {
	// Scenario: Match is 1-0. Home scores at 50' to make it 2-0.
	// However, VAR overturns the goal at 52'.
	// Then Away scores at 75' (1-1). Match ends 1-1.
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 20, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "valid_g1"},
		{Sequence: 2, Minute: 50, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "var_g2", IsOverturned: true}, // OVERTURNED
		{Sequence: 3, Minute: 75, ScoringTeam: "AWAY", HomeScoreAfter: 1, AwayScoreAfter: 1, EventID: "valid_a1"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_fixture_var",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 1, Away: 1},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Home should NOT win via 2UP because the 2-0 goal was overturned!
	// Match ended 1-1, so Home should be LOST.
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusLost {
		t.Errorf("Expected Home to be LOST after VAR reversal, got %s (reason: %s)", resHome.Status, resHome.Reason)
	}

	// Draw should be WON (FT 1-1)
	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusWon {
		t.Errorf("Expected Draw to be WON (1-1), got %s", resDraw.Status)
	}
}

func Test2UP_MissingTimeline_AmbiguousScore_FailClosedHold(t *testing.T) {
	// Scenario: FT score is 2-2. Timeline telemetry is MISSING (nil).
	// Home scored 2 goals. Home COULD have led 2-0 early.
	// Without timeline proof, marking Home LOST would be unsafe.
	// Evaluator MUST return SETTLEMENT_HOLD.
	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_ambiguous",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 2},
		EarlyPayout:        nil, // Missing telemetry!
		ScoreConfirmedAt:   time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	// Home bet on ambiguous score without timeline: MUST HOLD
	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusSettlementHold {
		t.Errorf("Expected BetStatusSettlementHold for Home on ambiguous score without timeline, got %s", resHome.Status)
	}
	if !resHome.DiscrepancyHeld {
		t.Errorf("Expected DiscrepancyHeld to be true")
	}

	// Away bet on ambiguous score without timeline: MUST HOLD (Away also scored 2 goals!)
	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusSettlementHold {
		t.Errorf("Expected BetStatusSettlementHold for Away on ambiguous score without timeline, got %s", resAway.Status)
	}

	// Draw bet is NOT ambiguous: FT is 2-2, Draw wins!
	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusWon {
		t.Errorf("Expected Draw to be WON on 2-2 FT, got %s", resDraw.Status)
	}
}

func Test2UP_MissingTimeline_UnambiguousScore_DefinitiveLoss(t *testing.T) {
	// Scenario: FT score is 0-1 (Away wins 1-0). Timeline telemetry is missing.
	// Home scored 0 goals. Home mathematically could NEVER have led by 2!
	// Therefore, Home is safely and definitively LOST.
	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_unambiguous",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 0, Away: 1},
		EarlyPayout:        nil,
		ScoreConfirmedAt:   time.Now().UTC(),
	}

	rule, _ := Get(1, "FB_1X2_2UP", PeriodFullTime, 1)

	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusLost {
		t.Errorf("Expected Home to be definitively LOST on 0-1 FT, got %s", resHome.Status)
	}

	// Away scored only 1 goal, could never have led by 2, BUT Away won FT (1-0), so Away is WON via FT fallback
	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_2UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusWon {
		t.Errorf("Expected Away to be WON via FT fallback, got %s", resAway.Status)
	}
}

func Test1UP_SettlementEvaluation(t *testing.T) {
	// Scenario: Match is 1-0 at FT (Home established 1-goal lead)
	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_1up",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 1, Away: 0},
		ScoreConfirmedAt:   time.Now().UTC(),
	}

	rule, ok := Get(1, "FB_1X2_1UP", PeriodFullTime, 1)
	if !ok {
		t.Fatalf("FB_1X2_1UP rule not registered")
	}

	resHome := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_1UP", Period: PeriodFullTime, SelectionCode: "1", RuleVersion: 1})
	if resHome.Status != BetStatusWon {
		t.Errorf("Expected 1UP Home to be WON, got %s (reason: %s)", resHome.Status, resHome.Reason)
	}
	if resHome.PayoutFactor != 1.0 {
		t.Errorf("Expected PayoutFactor 1.0, got %f", resHome.PayoutFactor)
	}

	resDraw := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_1UP", Period: PeriodFullTime, SelectionCode: "X", RuleVersion: 1})
	if resDraw.Status != BetStatusLost {
		t.Errorf("Expected 1UP Draw to be LOST on 1-0 FT, got %s", resDraw.Status)
	}

	resAway := rule.Evaluate(state, BetSelection{CanonicalMarketCode: "FB_1X2_1UP", Period: PeriodFullTime, SelectionCode: "2", RuleVersion: 1})
	if resAway.Status != BetStatusLost {
		t.Errorf("Expected 1UP Away to be LOST on 1-0 FT, got %s", resAway.Status)
	}
}

func Test2UP_ConcurrentIdempotentEvaluation(t *testing.T) {
	// Concurrent evaluation must be deterministic and thread-safe
	timeline := []GoalTimelineEntry{
		{Sequence: 1, Minute: 10, ScoringTeam: "HOME", HomeScoreAfter: 1, AwayScoreAfter: 0, EventID: "c1"},
		{Sequence: 2, Minute: 20, ScoringTeam: "HOME", HomeScoreAfter: 2, AwayScoreAfter: 0, EventID: "c2"},
		{Sequence: 3, Minute: 50, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 1, EventID: "c3"},
		{Sequence: 4, Minute: 70, ScoringTeam: "AWAY", HomeScoreAfter: 2, AwayScoreAfter: 2, EventID: "c4"},
	}

	state := CanonicalEventState{
		CanonicalFixtureID: "1:FB:test_concurrent",
		SportID:            1,
		Status:             "FT",
		IsFinished:         true,
		Score:              ScoreData{Home: 2, Away: 2},
		EarlyPayout: &EarlyPayoutTelemetry{
			GoalTimeline:     timeline,
			TimelineReliable: true,
		},
		ScoreConfirmedAt: time.Now().UTC(),
	}

	bet := BetSelection{
		BetID:               "bet_conc_1",
		SportID:             1,
		CanonicalMarketCode: "FB_1X2_2UP",
		Period:              PeriodFullTime,
		SelectionCode:       "1",
		RuleVersion:         1,
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := Evaluate(state, bet)
			if err != nil {
				t.Errorf("Unexpected error in Evaluate: %v", err)
			}
			if res.Status != BetStatusWon {
				t.Errorf("Concurrent eval expected WON, got %s", res.Status)
			}
		}()
	}
	wg.Wait()
}

func TestProviderMapping_EarlyPayouts(t *testing.T) {
	// SportyBet 60100 maps to FB_1X2_2UP (ACTIVE)
	res2UP := MapProviderMarket(ProviderMarketInput{
		Provider:       ProviderSportsXBet,
		SportID:        1,
		RawMarketID:    "60100",
		RawMarketName:  "1X2 - 2UP",
		RawOutcomeID:   "1",
		RawOutcomeName: "Home",
	})
	if !res2UP.Mapped {
		t.Fatalf("Expected 60100 to be mapped")
	}
	if res2UP.CanonicalMarketCode != "FB_1X2_2UP" {
		t.Errorf("Expected canonical code FB_1X2_2UP, got %s", res2UP.CanonicalMarketCode)
	}
	if res2UP.Status != MarketStatusActive {
		t.Errorf("Expected MarketStatusActive for FB_1X2_2UP, got %s (reason: %s)", res2UP.Status, res2UP.Reason)
	}

	// SportyBet 60200 maps to FB_1X2_1UP (ACTIVE)
	res1UP := MapProviderMarket(ProviderMarketInput{
		Provider:       ProviderSportsXBet,
		SportID:        1,
		RawMarketID:    "60200",
		RawMarketName:  "1X2 - 1UP",
		RawOutcomeID:   "1",
		RawOutcomeName: "Home",
	})
	if !res1UP.Mapped {
		t.Fatalf("Expected 60200 to be mapped")
	}
	if res1UP.CanonicalMarketCode != "FB_1X2_1UP" {
		t.Errorf("Expected canonical code FB_1X2_1UP, got %s", res1UP.CanonicalMarketCode)
	}
	if res1UP.Status != MarketStatusActive {
		t.Errorf("Expected MarketStatusActive for FB_1X2_1UP, got %s (reason: %s)", res1UP.Status, res1UP.Reason)
	}
}
