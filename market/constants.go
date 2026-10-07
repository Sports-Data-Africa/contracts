package market

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

// Market Operational Statuses
const (
	MarketStatusActive              = "ACTIVE"
	MarketStatusDisabledUnsupported = "DISABLED_UNSUPPORTED"
	MarketStatusSuspended           = "SUSPENDED"
)

// Active Provider Identifiers
const (
	ProviderSportyBet = "SPORTYBET"
	ProviderOdibets   = "ODIBETS"
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
