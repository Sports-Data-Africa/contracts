package market

import "strings"

// Canonical Result Domains
// A disagreement in DOMAIN_CORNERS must NOT prevent settlement of DOMAIN_SCORE markets.
const (
	DomainScore       = "DOMAIN_SCORE"
	DomainPeriodScore = "DOMAIN_PERIOD_SCORE"
	DomainCorners     = "DOMAIN_CORNERS"
	DomainCards       = "DOMAIN_CARDS"
	DomainPlayerStats = "DOMAIN_PLAYER_STATS"
	DomainTeamStats   = "DOMAIN_TEAM_STATS"
	DomainExtraTime   = "DOMAIN_EXTRA_TIME"
	DomainPenalties   = "DOMAIN_PENALTIES"
	DomainSets        = "DOMAIN_SETS"
	DomainGames       = "DOMAIN_GAMES"
	DomainPoints      = "DOMAIN_POINTS"
	DomainInnings     = "DOMAIN_INNINGS"
	DomainOvers       = "DOMAIN_OVERS"
	DomainRounds      = "DOMAIN_ROUNDS"
)

// Settlement Timing Model
const (
	// Outcome can become mathematically resolved while play continues (e.g. Over reached, Under busted, BTTS Yes/No locked)
	TimingInstantIrreversible = "INSTANT_IRREVERSIBLE"
	// Outcome must wait for the end of its applicable period unless mathematically locked earlier (e.g. HT 1X2, Q1 Winner, Tennis Set 1)
	TimingPeriodEnd = "PERIOD_END"
	// Outcome genuinely requires final event completion (e.g. FT 1X2, Correct Score)
	TimingEventEnd = "EVENT_END"
)

// Bet Statuses
const (
	BetStatusOpen                = "OPEN"
	BetStatusWon                 = "WON"
	BetStatusLost                = "LOST"
	BetStatusPush                = "PUSH"
	BetStatusVoid                = "VOID"
	BetStatusHalfWin             = "HALF_WIN"
	BetStatusHalfLoss            = "HALF_LOSS"
	BetStatusSettlementHold      = "SETTLEMENT_HOLD"
	BetStatusManualReview        = "MANUAL_REVIEW"
	BetStatusPendingConfirmation = "PENDING_CONFIRMATION"
)

// Fixture Statuses
const (
	FixtureStatusPrematch    = "PREMATCH"
	FixtureStatusLive        = "LIVE"
	FixtureStatusHT          = "HT"
	FixtureStatusLive2H      = "LIVE_2H"
	FixtureStatusFT          = "FT"
	FixtureStatusPostponed   = "POSTPONED"
	FixtureStatusAbandoned   = "ABANDONED"
	FixtureStatusCancelled   = "CANCELLED"
	FixtureStatusInterrupted = "INTERRUPTED"
)

// Market Operational & Capability Statuses
const (
	MarketStatusActive              = "ACTIVE"
	MarketStatusDisabledUnsupported = "DISABLED_UNSUPPORTED"
	MarketStatusSuspended           = "SUSPENDED"
	MarketStatusQuarantined         = "QUARANTINED"
	MarketStatusShadow              = "SHADOW"
)

// Canonical Formula Types (Mathematical models)
const (
	Formula1X2          = "1X2"
	FormulaMoneyline    = "MONEYLINE"
	FormulaTotals       = "TOTALS"
	FormulaHandicap     = "HANDICAP"
	FormulaBTTS         = "BTTS"
	FormulaDoubleChance = "DOUBLE_CHANCE"
	FormulaDNB          = "DNB"
	FormulaOddEven      = "ODD_EVEN"
	FormulaExact        = "EXACT"
)

// Canonical Metrics
const (
	MetricGoals   = "goals"
	MetricPoints  = "points"
	MetricCorners = "corners"
	MetricCards   = "cards"
	MetricSets    = "sets"
	MetricGames   = "games"
	MetricRounds  = "rounds"
	MetricTries   = "tries"
	MetricRuns    = "runs"
	MetricWickets = "wickets"
)

// Target Participants
const (
	TargetMatch = "match"
	TargetHome  = "home"
	TargetAway  = "away"
)

// Active Provider Identifiers
const (
	ProviderSportsXBet = "SPORTSXBET"
	ProviderOddixBets  = "ODDIXBETS"

	// Legacy aliases for backward compatibility and ingestion normalization
	ProviderSportyBet = "SPORTSXBET" // Deprecated: use ProviderSportsXBet
	ProviderOdibets   = "ODDIXBETS"  // Deprecated: use ProviderOddixBets
)

// Scope / Period Identifiers
const (
	PeriodFullTime   = "FT"
	PeriodHalfTime   = "HT"
	PeriodSecondHalf = "2H"
	PeriodExtraTime  = "ET"
	PeriodPenalties  = "PEN"
	PeriodQuarter1   = "Q1"
	PeriodQuarter2   = "Q2"
	PeriodQuarter3   = "Q3"
	PeriodQuarter4   = "Q4"
	PeriodSet1       = "SET1"
	PeriodSet2       = "SET2"
	PeriodSet3       = "SET3"
	PeriodSet4       = "SET4"
	PeriodSet5       = "SET5"
	PeriodPeriod1    = "PERIOD1"
	PeriodPeriod2    = "PERIOD2"
	PeriodPeriod3    = "PERIOD3"
	PeriodInnings1   = "INNINGS1"
	PeriodInnings2   = "INNINGS2"
	PeriodRound1     = "ROUND1"
	PeriodRound2     = "ROUND2"
	PeriodRound3     = "ROUND3"
	PeriodRound4     = "ROUND4"
	PeriodRound5     = "ROUND5"
)

// NormalizePeriod normalizes any variant period string into canonical Period constant.
func NormalizePeriod(period string) string {
	s := strings.ToUpper(strings.TrimSpace(period))
	switch s {
	case "FT", "FULL_TIME", "FULLTIME", "MATCH", "REGULAR":
		return PeriodFullTime
	case "HT", "HALF_TIME", "HALFTIME", "1H", "1ST_HALF", "FIRST_HALF":
		return PeriodHalfTime
	case "2H", "2ND_HALF", "SECOND_HALF":
		return PeriodSecondHalf
	case "ET", "EXTRA_TIME", "EXTRATIME":
		return PeriodExtraTime
	case "PEN", "PENALTIES", "PENALTY_SHOOTOUT":
		return PeriodPenalties
	case "Q1", "1ST_QUARTER", "QUARTER1", "QUARTER_1":
		return PeriodQuarter1
	case "Q2", "2ND_QUARTER", "QUARTER2", "QUARTER_2":
		return PeriodQuarter2
	case "Q3", "3RD_QUARTER", "QUARTER3", "QUARTER_3":
		return PeriodQuarter3
	case "Q4", "4TH_QUARTER", "QUARTER4", "QUARTER_4":
		return PeriodQuarter4
	case "SET1", "S1", "1ST_SET", "SET_1":
		return PeriodSet1
	case "SET2", "S2", "2ND_SET", "SET_2":
		return PeriodSet2
	case "SET3", "S3", "3RD_SET", "SET_3":
		return PeriodSet3
	case "SET4", "S4", "4TH_SET", "SET_4":
		return PeriodSet4
	case "SET5", "S5", "5TH_SET", "SET_5":
		return PeriodSet5
	case "PERIOD1", "P1", "1ST_PERIOD", "PERIOD_1":
		return PeriodPeriod1
	case "PERIOD2", "P2", "2ND_PERIOD", "PERIOD_2":
		return PeriodPeriod2
	case "PERIOD3", "P3", "3RD_PERIOD", "PERIOD_3":
		return PeriodPeriod3
	case "INNINGS1", "1ST_INNINGS":
		return PeriodInnings1
	case "INNINGS2", "2ND_INNINGS":
		return PeriodInnings2
	case "ROUND1", "R1", "1ST_ROUND":
		return PeriodRound1
	case "ROUND2", "R2", "2ND_ROUND":
		return PeriodRound2
	case "ROUND3", "R3", "3RD_ROUND":
		return PeriodRound3
	case "ROUND4", "R4", "4TH_ROUND":
		return PeriodRound4
	case "ROUND5", "R5", "5TH_ROUND":
		return PeriodRound5
	default:
		return s
	}
}

