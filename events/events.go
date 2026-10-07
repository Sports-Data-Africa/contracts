package events

import (
	"time"
)

// Commercial Feed Providers (Odds, Lines, Markets)
const (
	CommercialFeedSportybet = "SPORTYBET"
	CommercialFeedOdibets   = "ODIBETS"
	CommercialFeedParipesa  = "PARIPESA" // Deprecated legacy alias: quarantined from active ingestion
)

// Independent Truth / Verification Sources (Evidence, Quorum, Certification)
const (
	TruthSourceAiScore    = "AISCORE"
	TruthSourceFlashscore = "FLASHSCORE"
	TruthSourceSofaScore  = "SOFASCORE"
)

// Certification Statuses for Independent Settlement Protection
const (
	CertStatusUnverified          = "UNVERIFIED"
	CertStatusPendingVerification = "PENDING_VERIFICATION"
	CertStatusVerified            = "VERIFIED"
	CertStatusConflict            = "CONFLICT"
	CertStatusManualReview        = "MANUAL_REVIEW"
	CertStatusCertifiedFinal      = "CERTIFIED_FINAL"
)

// InternalMarketOddsUpdatedEvent contains complete operational provenance.
// STRICTLY INTERNAL: Must NEVER be serialized into B2B client responses.
type InternalMarketOddsUpdatedEvent struct {
	EventID            string              `json:"event_id"`
	CanonicalFixtureID string              `json:"canonical_fixture_id,omitempty"` // e.g. "SPD-FB-00982134"
	FixtureID          int64               `json:"fixture_id"`                      // SDA Numeric ID (legacy)
	SportID            int                 `json:"sport_id"`
	CommercialFeed     string              `json:"commercial_feed"` // SPORTYBET or ODIBETS
	FeedFixtureID      string              `json:"feed_fixture_id"` // Provider fixture ID
	MarketCode         string              `json:"market_code"`     // Canonical market code (e.g. "1X2")
	Period             string              `json:"period,omitempty"`
	LineType           string              `json:"line_type,omitempty"`
	MarketVersion      int64               `json:"market_version"` // Sovereign monotonic version
	IsSuspended        bool                `json:"is_suspended"`
	Selections         []InternalSelection `json:"selections"`
	UpstreamSequence   int64               `json:"upstream_sequence"`
	IngestedAt         int64               `json:"ingested_at"`
	PublishedAt        int64               `json:"published_at"`
}

// InternalSelection holds selection odds and raw upstream IDs for reconciliation.
type InternalSelection struct {
	Code            string  `json:"code"`              // Canonical code ("HOME", "DRAW", "AWAY")
	Odds            float64 `json:"odds"`              // Decimal odds (e.g. 1.95)
	FeedSelectionID string  `json:"feed_selection_id"` // Upstream provider selection ID
}

// InternalResultCertifiedEvent represents cryptographically verified match results.
// STRICTLY INTERNAL: Consumed by settlement-service to trigger deterministic payouts.
type InternalResultCertifiedEvent struct {
	EventID             string    `json:"event_id"`
	CanonicalFixtureID  string    `json:"canonical_fixture_id,omitempty"` // e.g. "SPD-FB-00982134"
	FixtureID           int64     `json:"fixture_id"`                      // SDA Numeric ID (legacy)
	SportID             int       `json:"sport_id"`
	HomeScore           int       `json:"home_score"`
	AwayScore           int       `json:"away_score"`
	PeriodScores        string    `json:"period_scores,omitempty"`
	CertificationStatus string    `json:"certification_status"` // CERTIFIED_FINAL, MANUAL_REVIEW, CONFLICT
	EvidenceHash        string    `json:"evidence_hash"`         // SHA-256 over scores + timestamp + quorum
	VerifiedAt          time.Time `json:"verified_at"`

	// STRICTLY INTERNAL AUDIT TRAIL: Never leak to clients
	VerificationSources []string `json:"verification_sources"` // ["AISCORE", "FLASHSCORE"]
}
