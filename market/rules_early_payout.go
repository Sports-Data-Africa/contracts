package market

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func init() {
	// Register Football Early Payout Rules
	Register(&Football2UPRule{})
	Register(&Football1UPRule{})
}

// ----------------------------------------------------------------------------
// 1X2 - 2UP Early Payout Rule (FB_1X2_2UP)
// ----------------------------------------------------------------------------

// Football2UPRule evaluates the 1X2 - 2UP Early Payout market (SportyBet raw 60100, Bet365 2UP).
//
// Promotion Specification:
//   - Backed selection "1" (Home Win): Wins early (INSTANT_IRREVERSIBLE) if the home team establishes
//     a 2-goal lead (e.g. 2-0, 3-1, 4-2) at any point during standard match play, regardless of the final score.
//   - Backed selection "2" (Away Win): Wins early (INSTANT_IRREVERSIBLE) if the away team establishes
//     a 2-goal lead (e.g. 0-2, 1-3, 2-4) at any point during standard match play, regardless of the final score.
//   - Backed selection "X" (Draw): Does NOT qualify for early payout; only settles at full-time final score.
//   - Full-Time Fallback: If neither team established a 2-goal lead, selections settle according to the
//     normal full-time 1X2 final score.
//
// Telemetry & Safety Safeguards:
//   - Ordered goal timeline tracking validates the exact score progression.
//   - VAR reversals that overturn the qualifying goal before official confirmation invalidate the lead.
//   - Missing or incomplete timeline telemetry triggers fail-closed SETTLEMENT_HOLD if the backed team scored
//     >= 2 goals at full-time, preventing false-loss settlements when history cannot be mathematically proven.
type Football2UPRule struct{}

func (r *Football2UPRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_1X2_2UP", Period: PeriodFullTime, RuleVersion: 1}
}

func (r *Football2UPRule) RequiredDomain() string { return DomainScore }
func (r *Football2UPRule) Timing() string         { return TimingInstantIrreversible }

// ReconstructTimeline processes ordered goal events, handles duplicates and VAR reversals,
// and determines if a 2-goal lead was ever established by either team.
func ReconstructTimeline(entries []GoalTimelineEntry) (home2Up bool, away2Up bool, homeMin int, awayMin int, reliable bool) {
	if len(entries) == 0 {
		return false, false, 0, 0, false
	}

	// Sort chronologically by Sequence and Minute
	sorted := make([]GoalTimelineEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sequence != sorted[j].Sequence {
			return sorted[i].Sequence < sorted[j].Sequence
		}
		return sorted[i].Minute < sorted[j].Minute
	})

	seenEvents := make(map[string]bool)
	homeRunning := 0
	awayRunning := 0

	for _, g := range sorted {
		if g.EventID != "" {
			if seenEvents[g.EventID] {
				// Deduplicate duplicate provider event emission
				continue
			}
			seenEvents[g.EventID] = true
		}

		if g.IsOverturned {
			// Overturned by VAR: does not count towards lead
			continue
		}

		team := strings.ToUpper(strings.TrimSpace(g.ScoringTeam))
		switch team {
		case "HOME", "1":
			homeRunning++
		case "AWAY", "2":
			awayRunning++
		default:
			// If scoring team is missing, infer from score delta
			if g.HomeScoreAfter > homeRunning {
				homeRunning = g.HomeScoreAfter
			}
			if g.AwayScoreAfter > awayRunning {
				awayRunning = g.AwayScoreAfter
			}
		}

		// Check lead differential
		if !home2Up && (homeRunning-awayRunning >= 2) {
			home2Up = true
			homeMin = g.Minute
		}
		if !away2Up && (awayRunning-homeRunning >= 2) {
			away2Up = true
			awayMin = g.Minute
		}
	}

	return home2Up, away2Up, homeMin, awayMin, true
}

func (r *Football2UPRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	switch sel {
	case "HOME", "HOME_WIN":
		sel = "1"
	case "DRAW":
		sel = "X"
	case "AWAY", "AWAY_WIN":
		sel = "2"
	}

	if sel != "1" && sel != "X" && sel != "2" {
		return SettlementResult{
			Status:         BetStatusManualReview,
			Settled:        false,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Invalid selection code '%s' for 1X2 2UP", bet.SelectionCode),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	// 1. Check for Early Payout Qualification via Telemetry or Timeline
	home2Up := false
	away2Up := false
	timelineReliable := false
	var triggerMin int

	if state.EarlyPayout != nil {
		if state.EarlyPayout.HomeTwoGoalLeadAchieved {
			home2Up = true
			triggerMin = state.EarlyPayout.HomeLeadTriggerMinute
		}
		if state.EarlyPayout.AwayTwoGoalLeadAchieved {
			away2Up = true
			triggerMin = state.EarlyPayout.AwayLeadTriggerMinute
		}
		timelineReliable = state.EarlyPayout.TimelineReliable

		// Reconstruct and verify against raw goal timeline if available
		if len(state.EarlyPayout.GoalTimeline) > 0 {
			recHome, recAway, recHomeMin, recAwayMin, recReliable := ReconstructTimeline(state.EarlyPayout.GoalTimeline)
			if recReliable {
				home2Up = recHome
				away2Up = recAway
				if home2Up && triggerMin == 0 {
					triggerMin = recHomeMin
				}
				if away2Up && triggerMin == 0 {
					triggerMin = recAwayMin
				}
				timelineReliable = true
			}
		}
	}

	// 2. Instant Win Evaluation: Backed team reached 2-goal lead
	if sel == "1" && home2Up {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Early payout 2UP triggered: Home established 2-goal lead at minute %d (Irreversible win)", triggerMin),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	if sel == "2" && away2Up {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Early payout 2UP triggered: Away established 2-goal lead at minute %d (Irreversible win)", triggerMin),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	// Draw ("X") NEVER settles early on 2UP
	// If the event is not finished, non-triggered bets remain OPEN
	if !state.IsFinished {
		return SettlementResult{
			Status:         BetStatusOpen,
			Settled:        false,
			Timing:         TimingInstantIrreversible,
			Reason:         "Match in-play; 2-goal lead early payout condition not yet satisfied",
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	// 3. Match Is Finished — Fallback to Full-Time 1X2 Settlement with Timeline Integrity Guard
	ftHome := state.Score.Home
	ftAway := state.Score.Away

	// Ambiguity / Missing Timeline Guard:
	// If the backed team scored >= 2 goals at full-time, but did not win at FT (e.g. 2-2, 2-3, 3-3),
	// they COULD have led 2-0 earlier in the match.
	// If timeline data is missing or unreliable, we MUST NOT mark them LOST without authoritative sequence proof.
	if !timelineReliable {
		if sel == "1" && ftHome >= 2 && ftHome <= ftAway {
			return SettlementResult{
				Status:          BetStatusSettlementHold,
				Settled:         false,
				Timing:          TimingEventEnd,
				Reason:          fmt.Sprintf("Timeline telemetry unavailable or incomplete for FT score %d-%d. Home scored %d goals and may have satisfied 2UP early lead; held for audit verification", ftHome, ftAway, ftHome),
				RequiredDomain:  DomainScore,
				DiscrepancyHeld: true,
				EvaluatedAt:     time.Now().UTC(),
			}
		}
		if sel == "2" && ftAway >= 2 && ftAway <= ftHome {
			return SettlementResult{
				Status:          BetStatusSettlementHold,
				Settled:         false,
				Timing:          TimingEventEnd,
				Reason:          fmt.Sprintf("Timeline telemetry unavailable or incomplete for FT score %d-%d. Away scored %d goals and may have satisfied 2UP early lead; held for audit verification", ftHome, ftAway, ftAway),
				RequiredDomain:  DomainScore,
				DiscrepancyHeld: true,
				EvaluatedAt:     time.Now().UTC(),
			}
		}
	}

	// Deterministic Full-Time 1X2 Resolution
	switch sel {
	case "1":
		if ftHome > ftAway {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Home won %d-%d (2UP fallback)", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Home did not win (%d-%d) and never established 2-goal lead", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}

	case "2":
		if ftAway > ftHome {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Away won %d-%d (2UP fallback)", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Away did not win (%d-%d) and never established 2-goal lead", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}

	case "X":
		if ftHome == ftAway {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Match drawn %d-%d", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Match did not draw (%d-%d)", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	return SettlementResult{
		Status:         BetStatusManualReview,
		Settled:        false,
		Timing:         TimingEventEnd,
		Reason:         "Unhandled selection in 2UP evaluation",
		RequiredDomain: DomainScore,
		EvaluatedAt:    time.Now().UTC(),
	}
}

// ----------------------------------------------------------------------------
// 1X2 - 1UP Early Payout Rule (FB_1X2_1UP) — ACTIVE & RELEASED
// ----------------------------------------------------------------------------

// Reconstruct1UPTimeline processes ordered goal events, handles duplicates and VAR reversals,
// and determines if a 1-goal lead was ever established by either team.
func Reconstruct1UPTimeline(entries []GoalTimelineEntry) (home1Up bool, away1Up bool, homeMin int, awayMin int, reliable bool) {
	if len(entries) == 0 {
		return false, false, 0, 0, false
	}

	sorted := make([]GoalTimelineEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sequence != sorted[j].Sequence {
			return sorted[i].Sequence < sorted[j].Sequence
		}
		return sorted[i].Minute < sorted[j].Minute
	})

	seenEvents := make(map[string]bool)
	homeRunning := 0
	awayRunning := 0

	for _, g := range sorted {
		if g.EventID != "" {
			if seenEvents[g.EventID] {
				continue
			}
			seenEvents[g.EventID] = true
		}

		if g.IsOverturned {
			continue
		}

		team := strings.ToUpper(strings.TrimSpace(g.ScoringTeam))
		switch team {
		case "HOME", "1":
			homeRunning++
		case "AWAY", "2":
			awayRunning++
		default:
			if g.HomeScoreAfter > homeRunning {
				homeRunning = g.HomeScoreAfter
			}
			if g.AwayScoreAfter > awayRunning {
				awayRunning = g.AwayScoreAfter
			}
		}

		if !home1Up && (homeRunning-awayRunning >= 1) {
			home1Up = true
			homeMin = g.Minute
		}
		if !away1Up && (awayRunning-homeRunning >= 1) {
			away1Up = true
			awayMin = g.Minute
		}
	}

	return home1Up, away1Up, homeMin, awayMin, true
}

// Football1UPRule represents the 1X2 - 1UP Early Payout promotional market (SportyBet raw 60200).
//
// Promotion Specification:
//   - Backed selection "1" (Home Win): Wins early (INSTANT_IRREVERSIBLE) if Home establishes a 1-goal lead.
//   - Backed selection "2" (Away Win): Wins early (INSTANT_IRREVERSIBLE) if Away establishes a 1-goal lead.
//   - Backed selection "X" (Draw): Resolves at full-time final score.
type Football1UPRule struct{}

func (r *Football1UPRule) Key() MarketRuleKey {
	return MarketRuleKey{SportID: 1, MarketCode: "FB_1X2_1UP", Period: PeriodFullTime, RuleVersion: 1}
}

func (r *Football1UPRule) RequiredDomain() string { return DomainScore }
func (r *Football1UPRule) Timing() string         { return TimingInstantIrreversible }

func (r *Football1UPRule) Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult {
	sel := strings.ToUpper(strings.TrimSpace(bet.SelectionCode))

	switch sel {
	case "HOME", "HOME_WIN":
		sel = "1"
	case "DRAW":
		sel = "X"
	case "AWAY", "AWAY_WIN":
		sel = "2"
	}

	if sel != "1" && sel != "X" && sel != "2" {
		return SettlementResult{
			Status:         BetStatusManualReview,
			Settled:        false,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Invalid selection code '%s' for 1X2 1UP", bet.SelectionCode),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	home1Up := false
	away1Up := false
	var triggerMin int

	if state.EarlyPayout != nil && len(state.EarlyPayout.GoalTimeline) > 0 {
		recHome, recAway, recHomeMin, recAwayMin, recReliable := Reconstruct1UPTimeline(state.EarlyPayout.GoalTimeline)
		if recReliable {
			home1Up = recHome
			away1Up = recAway
			if home1Up {
				triggerMin = recHomeMin
			} else if away1Up {
				triggerMin = recAwayMin
			}
		}
	} else {
		// Fallback to score if timeline unavailable
		if state.Score.Home > 0 && state.Score.Home > state.Score.Away {
			home1Up = true
		} else if state.Score.Away > 0 && state.Score.Away > state.Score.Home {
			away1Up = true
		}
	}

	// 1. Early win trigger
	if sel == "1" && home1Up {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Early payout 1UP triggered: Home established 1-goal lead at minute %d (Irreversible win)", triggerMin),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	if sel == "2" && away1Up {
		return SettlementResult{
			Status:         BetStatusWon,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingInstantIrreversible,
			Reason:         fmt.Sprintf("Early payout 1UP triggered: Away established 1-goal lead at minute %d (Irreversible win)", triggerMin),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	// Draw ("X") NEVER settles early on 1UP
	if !state.IsFinished {
		return SettlementResult{
			Status:         BetStatusOpen,
			Settled:        false,
			Timing:         TimingInstantIrreversible,
			Reason:         "Match in-play; 1-goal lead early payout condition not yet satisfied",
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	// 2. Full-time fallback
	ftHome := state.Score.Home
	ftAway := state.Score.Away

	switch sel {
	case "1":
		if ftHome > ftAway {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Home won %d-%d (1UP fallback)", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Home did not win (%d-%d) and never established 1-goal lead", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	case "2":
		if ftAway > ftHome {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Away won %d-%d (1UP fallback)", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Away did not win (%d-%d) and never established 1-goal lead", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	case "X":
		if ftHome == ftAway {
			return SettlementResult{
				Status:         BetStatusWon,
				PayoutFactor:   1.0,
				Settled:        true,
				Timing:         TimingEventEnd,
				Reason:         fmt.Sprintf("FT Result Win: Match drawn %d-%d", ftHome, ftAway),
				RequiredDomain: DomainScore,
				EvaluatedAt:    time.Now().UTC(),
			}
		}
		return SettlementResult{
			Status:         BetStatusLost,
			PayoutFactor:   0.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         fmt.Sprintf("FT Result Loss: Match did not draw (%d-%d)", ftHome, ftAway),
			RequiredDomain: DomainScore,
			EvaluatedAt:    time.Now().UTC(),
		}
	}

	return SettlementResult{
		Status:         BetStatusManualReview,
		Settled:        false,
		Timing:         TimingEventEnd,
		Reason:         "Unhandled selection in 1UP evaluation",
		RequiredDomain: DomainScore,
		EvaluatedAt:    time.Now().UTC(),
	}
}
