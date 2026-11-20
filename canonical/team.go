package canonical

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidCanonicalTeamKey = errors.New("canonical: invalid canonical team key")
	ErrInvalidCanonicalCompKey = errors.New("canonical: invalid canonical competition key")
)

// TeamLogoReference encapsulates logo provenance and verification metadata
// without storing any image binaries on internal servers.
type TeamLogoReference struct {
	URL             string     `json:"url,omitempty"`                // Primary verified public CDN URL (SVG or PNG)
	FallbackURL     string     `json:"fallback_url,omitempty"`       // Secondary high-res PNG fallback URL
	ProviderLogoURL string     `json:"provider_logo_url,omitempty"` // Original upstream wire icon URL
	Provenance      string     `json:"provenance,omitempty"`        // Source tag (e.g., 'wikimedia', 'nba_cdn', 'provider')
	Status          string     `json:"status,omitempty"`            // 'VERIFIED', 'UNVERIFIED', 'FAILED'
	FallbackType    string     `json:"fallback_type,omitempty"`     // 'none', 'monogram', 'provider'
	HTTPStatus      int        `json:"http_status,omitempty"`       // Last probed HTTP status (e.g., 200, 404)
	LastVerifiedAt  *time.Time `json:"last_verified_at,omitempty"`  // Timestamp of last active HTTP probe
}

// CanonicalTeam represents an authoritative, sovereign real-world sports competitor.
type CanonicalTeam struct {
	ID           int64             `json:"id"`                      // Internal primary key
	CanonicalKey string            `json:"canonical_key"`           // Sovereign SPDA ID (e.g. "SDA-FB-100001")
	SportID      int               `json:"sport_id"`                // Aligned with canonical sport IDs (1=FB, 3=BB, etc.)
	SportCode    string            `json:"sport_code"`              // "FB", "BB", "TN", "IH", etc.
	Name         string            `json:"name"`                    // Official full name (e.g. "Manchester City")
	ShortName    string            `json:"short_name,omitempty"`    // Monogram / TLA (e.g. "MCI")
	CountryCode  string            `json:"country_code,omitempty"`  // ISO 3166-1 alpha-2 (e.g. "gb-eng", "ke")
	Gender       string            `json:"gender,omitempty"`        // "MEN", "WOMEN", "MIXED"
	AgeCategory  string            `json:"age_category,omitempty"`  // "SENIOR", "U21", "U19", "YOUTH"
	Logo         TeamLogoReference `json:"logo"`                    // Zero-storage public CDN & provider logo reference
	Status       string            `json:"status"`                  // "ACTIVE", "INACTIVE", "PENDING_REVIEW"
	IsVerified   bool              `json:"is_verified"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// TeamReference is the compact team object embedded in public B2B fixture feeds.
type TeamReference struct {
	ID          int64             `json:"id"`
	CanonicalID string            `json:"canonical_id"` // Sovereign SPDA ID
	Name        string            `json:"name"`
	ShortName   string            `json:"short_name,omitempty"`
	CountryCode string            `json:"country_code,omitempty"`
	Logo        TeamLogoReference `json:"logo"`
}

// CompetitionLogoReference encapsulates competition/league logo endpoints.
type CompetitionLogoReference struct {
	URL             string     `json:"url,omitempty"`
	FallbackURL     string     `json:"fallback_url,omitempty"`
	ProviderLogoURL string     `json:"provider_logo_url,omitempty"`
	Provenance      string     `json:"provenance,omitempty"`
	Status          string     `json:"status,omitempty"`
	LastVerifiedAt  *time.Time `json:"last_verified_at,omitempty"`
}

// CanonicalCompetition represents an authoritative league or tournament.
type CanonicalCompetition struct {
	ID           int64                    `json:"id"`
	CanonicalKey string                   `json:"canonical_key"` // Sovereign key, e.g. "SDA-COMP-100201"
	SportID      int                      `json:"sport_id"`
	SportCode    string                   `json:"sport_code"`
	Name         string                   `json:"name"`                   // e.g. "Premier League"
	CountryCode  string                   `json:"country_code,omitempty"` // "gb-eng"
	CountryName  string                   `json:"country_name,omitempty"` // "England"
	Logo         CompetitionLogoReference `json:"logo"`
	Status       string                   `json:"status"`                 // "ACTIVE", "INACTIVE"
	IsVerified   bool                     `json:"is_verified"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// CompetitionReference is the compact competition object embedded in public B2B fixture feeds.
type CompetitionReference struct {
	ID          int64                    `json:"id"`
	CanonicalID string                   `json:"canonical_id"`
	Name        string                   `json:"name"`
	CountryCode string                   `json:"country_code,omitempty"`
	CountryName string                   `json:"country_name,omitempty"`
	Logo        CompetitionLogoReference `json:"logo"`
}

// FormatCanonicalTeamKey formats a sovereign immutable SPDA team key.
// E.g., SportID=1, ID=100001 -> "SPD-FB-100001"
func FormatCanonicalTeamKey(sportID int, spdaTeamID int64) string {
	code, ok := SportIDToCode[sportID]
	if !ok || code == "" {
		code = "GEN"
	}
	return fmt.Sprintf("SPD-%s-%d", code, spdaTeamID)
}

// ParseCanonicalTeamKey parses a sovereign team key back to sport code and numeric ID.
// Accepts both current "SPD-" and legacy "SDA-" prefixes for robust backwards compatibility.
func ParseCanonicalTeamKey(key string) (string, int64, error) {
	parts := strings.Split(strings.TrimSpace(key), "-")
	if len(parts) != 3 || (parts[0] != "SPD" && parts[0] != "SDA") {
		return "", 0, fmt.Errorf("%w: %q", ErrInvalidCanonicalTeamKey, key)
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, fmt.Errorf("%w: invalid numeric id in %q", ErrInvalidCanonicalTeamKey, key)
	}
	return parts[1], id, nil
}

// FormatCanonicalCompetitionKey formats a sovereign immutable competition key.
// E.g., ID=100201 -> "SPD-COMP-100201"
func FormatCanonicalCompetitionKey(compID int64) string {
	return fmt.Sprintf("SPD-COMP-%d", compID)
}

// ParseCanonicalCompetitionKey parses an immutable competition key.
// Accepts both current "SPD-" and legacy "SDA-" prefixes.
func ParseCanonicalCompetitionKey(key string) (int64, error) {
	parts := strings.Split(strings.TrimSpace(key), "-")
	if len(parts) != 3 || (parts[0] != "SPD" && parts[0] != "SDA") || parts[1] != "COMP" {
		return 0, fmt.Errorf("%w: %q", ErrInvalidCanonicalCompKey, key)
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: invalid numeric id in %q", ErrInvalidCanonicalCompKey, key)
	}
	return id, nil
}
