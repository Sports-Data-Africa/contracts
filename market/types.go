package market

import (
	"fmt"
	"time"
)

// MarketRuleKey identifies a unique canonical settlement rule version.
type MarketRuleKey struct {
	SportID     int    `json:"sport_id"`
	MarketCode  string `json:"market_code"`
	Period      string `json:"period"`
	RuleVersion int    `json:"rule_version"`
}

func (k MarketRuleKey) String() string {
	return fmt.Sprintf("SP_%d:%s:%s:v%d", k.SportID, k.MarketCode, k.Period, k.RuleVersion)
}

// MarketDefinition represents the canonical blueprint for a customer-facing betting market.
type MarketDefinition struct {
	SportID              int      `json:"sport_id"`
	MarketCode           string   `json:"market_code"`
	MarketName           string   `json:"market_name"`
	Period               string   `json:"period"`
	RequiredResultDomain string   `json:"required_result_domain"`
	SettlementTiming     string   `json:"settlement_timing"`
	RuleVersion          int      `json:"rule_version"`
	SupportedSelections  []string `json:"supported_selections"`
	HasLine              bool     `json:"has_line"`
	LineType             string   `json:"line_type,omitempty"` // "DECIMAL", "QUARTERS", "INTEGER"
	IncludesOvertime     bool     `json:"includes_overtime"`
	IncludesShootout     bool     `json:"includes_shootout"`
	Status               string   `json:"status"` // ACTIVE, DISABLED_UNSUPPORTED
}

// ScoreData captures a 2-way home/away score.
type ScoreData struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

func (s ScoreData) Total() int {
	return s.Home + s.Away
}

// StatData captures statistical metrics (e.g. corners).
type StatData struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

func (s StatData) Total() int {
	return s.Home + s.Away
}

// CardStatData captures discipline counts.
type CardStatData struct {
	HomeYellow int `json:"home_yellow"`
	AwayYellow int `json:"away_yellow"`
	HomeRed    int `json:"home_red"`
	AwayRed    int `json:"away_red"`
}

// TennisSetData captures tennis set and tiebreak state.
type TennisSetData struct {
	SetNumber          int  `json:"set_number"`
	HomeGames          int  `json:"home_games"`
	AwayGames          int  `json:"away_games"`
	HomeTiebreakPoints int  `json:"home_tiebreak_points,omitempty"`
	AwayTiebreakPoints int  `json:"away_tiebreak_points,omitempty"`
	IsFinished         bool `json:"is_finished"`
}

// CricketInningsData captures cricket innings runs, wickets, and overs.
type CricketInningsData struct {
	InningsNumber int     `json:"innings_number"`
	BattingTeam   string  `json:"batting_team"`
	Runs          int     `json:"runs"`
	Wickets       int     `json:"wickets"`
	Overs         float64 `json:"overs"`
	IsCompleted   bool    `json:"is_completed"`
}

// CricketStateData captures the full state required for cricket settlement.
type CricketStateData struct {
	Format         string               `json:"format"` // TEST, ODI, T20, T10, 100-BALL
	Innings        []CricketInningsData `json:"innings"`
	CurrentInnings int                  `json:"current_innings"`
	TargetRuns     int                  `json:"target_runs,omitempty"`
	ScheduledOvers int                  `json:"scheduled_overs"`
	RevisedOvers   int                  `json:"revised_overs,omitempty"`
	DLSApplied     bool                 `json:"dls_applied"`
	SuperOverScore *ScoreData           `json:"super_over_score,omitempty"`
	MatchCompleted bool                 `json:"match_completed"`
	Winner         string               `json:"winner"` // "1", "2", "DRAW", "TIE", "NO_RESULT"
	Fours          StatData             `json:"fours"`
	Sixes          StatData             `json:"sixes"`
}

// BaseballInningData captures single baseball inning scores.
type BaseballInningData struct {
	InningNumber int  `json:"inning_number"`
	HomeRuns     int  `json:"home_runs"`
	AwayRuns     int  `json:"away_runs"`
	IsFinished   bool `json:"is_finished"`
}

// BaseballStateData captures baseball-specific innings and match completion state.
type BaseballStateData struct {
	Innings              []BaseballInningData `json:"innings"`
	CurrentInning        int                  `json:"current_inning"`
	IsTopInning          bool                 `json:"is_top_inning"`
	HitsHome             int                  `json:"hits_home"`
	HitsAway             int                  `json:"hits_away"`
	ErrorsHome           int                  `json:"errors_home"`
	ErrorsAway           int                  `json:"errors_away"`
	MatchCompleted       bool                 `json:"match_completed"`
	IncompleteGameCalled bool                 `json:"incomplete_game_called"`
}

// MMAStateData captures fight classification.
type MMAStateData struct {
	CompletedRounds         int    `json:"completed_rounds"`
	ScheduledRounds         int    `json:"scheduled_rounds"`
	MethodOfVictory         string `json:"method_of_victory"` // "KO", "TKO", "SUBMISSION", "DECISION_UNANIMOUS", "DECISION_SPLIT", "DRAW", "NO_CONTEST", "DISQUALIFICATION"
	WinningFighter          string `json:"winning_fighter"`   // "1", "2", "DRAW", "NO_CONTEST"
	EndRound                int    `json:"end_round"`
	EndTimeSeconds          int    `json:"end_time_seconds"`
	OfficialResultCertified bool   `json:"official_result_certified"`
}

// ProviderEvidenceData retains audit evidence for traceability back to provider messages.
type ProviderEvidenceData struct {
	Provider          string    `json:"provider"`
	ProviderEventID   string    `json:"provider_event_id"`
	Domain            string    `json:"domain"`
	PayloadHash       string    `json:"payload_hash"`
	ProviderTimestamp time.Time `json:"provider_timestamp"`
	ReceivedTimestamp time.Time `json:"received_timestamp"`
	Sequence          int64     `json:"sequence"`
}

// CanonicalEventState is the normalized event state holding domain-separated verified data.
type CanonicalEventState struct {
	CanonicalFixtureID string                    `json:"canonical_fixture_id"`
	SportID            int                       `json:"sport_id"`
	Status             string                    `json:"status"` // PREMATCH, LIVE, HT, FT, POSTPONED, etc.
	IsFinished         bool                      `json:"is_finished"`
	IsPeriodFinished   map[string]bool           `json:"is_period_finished"`
	CurrentPeriod      string                    `json:"current_period"`
	Score              ScoreData                 `json:"score"`
	PeriodScores       map[string]ScoreData      `json:"period_scores"`
	Corners            map[string]StatData       `json:"corners"`
	Cards              map[string]CardStatData   `json:"cards"`
	PlayerStats        map[string]map[string]int `json:"player_stats"`
	TeamStats          map[string]map[string]int `json:"team_stats"`
	ExtraTime          *ScoreData                `json:"extra_time,omitempty"`
	Penalties          *ScoreData                `json:"penalties,omitempty"`
	TennisSets         []TennisSetData           `json:"tennis_sets,omitempty"`
	TennisRetired      bool                      `json:"tennis_retired,omitempty"`
	TennisWalkover     bool                      `json:"tennis_walkover,omitempty"`
	Cricket            *CricketStateData         `json:"cricket,omitempty"`
	Baseball           *BaseballStateData        `json:"baseball,omitempty"`
	MMA                *MMAStateData             `json:"mma,omitempty"`
	ProviderEvidence   []ProviderEvidenceData    `json:"provider_evidence"`
	ScoreConfirmedAt   time.Time                 `json:"score_confirmed_at"`
	LastUpdatedAt      time.Time                 `json:"last_updated_at"`
}

// BetSelection is the immutable snapshot stored at bet placement time.
type BetSelection struct {
	BetID               string    `json:"bet_id"`
	LegID               string    `json:"leg_id,omitempty"`
	CanonicalFixtureID  string    `json:"canonical_fixture_id"`
	SportID             int       `json:"sport_id"`
	CanonicalMarketCode string    `json:"canonical_market_code"`
	Period              string    `json:"period"`
	SelectionCode       string    `json:"selection_code"`
	SelectionName       string    `json:"selection_name,omitempty"`
	Line                *float64  `json:"line,omitempty"`
	LineQuarters        *int      `json:"line_quarters,omitempty"` // Exact integer quarters: 0.25 -> 1, 0.50 -> 2, 0.75 -> 3, 1.00 -> 4, -0.25 -> -1
	Provider            string    `json:"provider"`
	ProviderMarketID    string    `json:"provider_market_id"`
	ProviderOutcomeID   string    `json:"provider_outcome_id"`
	OddsAtPlacement     float64   `json:"odds_at_placement"`
	Stake               float64   `json:"stake"`
	RuleVersion         int       `json:"rule_version"`
	PlacedAt            time.Time `json:"placed_at"`
}

// SettlementResult represents the deterministic evaluation result of a bet selection.
type SettlementResult struct {
	Status          string    `json:"status"`        // WON, LOST, PUSH, VOID, HALF_WIN, HALF_LOSS, OPEN, SETTLEMENT_HOLD, MANUAL_REVIEW
	PayoutFactor    float64   `json:"payout_factor"` // 1.0 (full win), 0.5 (half win), 0.0 (loss), 1.0 (void/push refund)
	Settled         bool      `json:"settled"`       // true if outcome has resolved, false if still in-progress
	Timing          string    `json:"timing"`        // INSTANT_IRREVERSIBLE, PERIOD_END, EVENT_END
	Reason          string    `json:"reason"`
	EvaluatedAt     time.Time `json:"evaluated_at"`
	RequiredDomain  string    `json:"required_domain"`
	DiscrepancyHeld bool      `json:"discrepancy_held,omitempty"`
}

// SettlementRule is the clean rule abstraction for evaluating a bet against canonical state.
type SettlementRule interface {
	Key() MarketRuleKey
	RequiredDomain() string
	Timing() string
	Evaluate(state CanonicalEventState, bet BetSelection) SettlementResult
}
