package identity

import (
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

// CanonicalFixture represents one real-world sporting event in the platform.
type CanonicalFixture struct {
	ID                  int64                          `json:"id"`
	CanonicalFixtureID  string                         `json:"canonical_fixture_id"` // e.g. SPD-FB-456778
	SequenceNumber      int64                          `json:"sequence_number"`      // e.g. 456778
	SportID             int                            `json:"sport_id"`
	SportCode           string                         `json:"sport_code"`
	HomeTeam            string                         `json:"home_team"`
	AwayTeam            string                         `json:"away_team"`
	ScheduledStart      time.Time                      `json:"scheduled_start"`
	CompetitionID       int64                          `json:"competition_id"`
	CompetitionName     string                         `json:"competition_name"`
	NaturalIdentityHash string                         `json:"natural_identity_hash"`
	IdentityState       canonical.IdentityState        `json:"identity_state"`
	VerificationState   canonical.VerificationState    `json:"verification_state"`
	FixtureState        canonical.FixtureLifecycleState `json:"fixture_state"`
	CreatedAt           time.Time                      `json:"created_at"`
	UpdatedAt           time.Time                      `json:"updated_at"`
}

// FixtureAlias maps an external provider fixture identifier to the platform's canonical identity.
type FixtureAlias struct {
	ID                 int64                 `json:"id"`
	CanonicalFixtureID string                `json:"canonical_fixture_id"`
	Provider           string                `json:"provider"`
	ProviderFixtureID  string                `json:"provider_fixture_id"`
	MappingType        canonical.MappingType `json:"mapping_type"` // PREMATCH, LIVE, RESULT
	Source             string                `json:"source"`
	FirstSeenAt        time.Time             `json:"first_seen_at"`
	LastSeenAt         time.Time             `json:"last_seen_at"`
	MetadataJSON       string                `json:"metadata_json,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

// IdentityConflict logs any attempts to map an existing provider fixture to a conflicting canonical fixture.
type IdentityConflict struct {
	ID                  int64     `json:"id"`
	ConflictType        string    `json:"conflict_type"` // e.g. ALIAS_REASSIGNMENT_ATTEMPT
	Provider            string    `json:"provider"`
	ProviderFixtureID   string    `json:"provider_fixture_id"`
	ExistingCanonicalID string    `json:"existing_canonical_id"`
	ProposedCanonicalID string    `json:"proposed_canonical_id"`
	EvidenceJSON        string    `json:"evidence_json"`
	Status              string    `json:"status"` // UNRESOLVED, RESOLVED, DISMISSED
	CreatedAt           time.Time `json:"created_at"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
}
