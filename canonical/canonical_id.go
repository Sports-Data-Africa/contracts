package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Canonical Prefix
const Prefix = "SPD"

// Standard Sport Codes
const (
	SportCodeFootball    = "FB"
	SportCodeIceHockey   = "IH"
	SportCodeBasketball  = "BB"
	SportCodeTennis      = "TN"
	SportCodeVolleyball  = "VB"
	SportCodeRugby       = "RB"
	SportCodeMMA         = "MM"
	SportCodeCricket     = "CK"
	SportCodeTableTennis = "TT" // Internal Quarantine Only
)

// Sport IDs
const (
	SportIDFootball    = 1
	SportIDIceHockey   = 2
	SportIDBasketball  = 3
	SportIDTennis      = 4
	SportIDVolleyball  = 6
	SportIDRugby       = 7
	SportIDMMA         = 9
	SportIDCricket     = 66
	SportIDTableTennis = 10
)

// Mapping from numeric Sport ID to 2-letter Sport Code
var SportIDToCode = map[int]string{
	SportIDFootball:    SportCodeFootball,
	SportIDIceHockey:   SportCodeIceHockey,
	SportIDBasketball:  SportCodeBasketball,
	SportIDTennis:      SportCodeTennis,
	SportIDVolleyball:  SportCodeVolleyball,
	SportIDRugby:       SportCodeRugby,
	SportIDMMA:         SportCodeMMA,
	SportIDCricket:     SportCodeCricket,
	SportIDTableTennis: SportCodeTableTennis,
}

// Mapping from 2-letter Sport Code to numeric Sport ID
var SportCodeToID = map[string]int{
	SportCodeFootball:    SportIDFootball,
	SportCodeIceHockey:   SportIDIceHockey,
	SportCodeBasketball:  SportIDBasketball,
	SportCodeTennis:      SportIDTennis,
	SportCodeVolleyball:  SportIDVolleyball,
	SportCodeRugby:       SportIDRugby,
	SportCodeMMA:         SportIDMMA,
	SportCodeCricket:     SportIDCricket,
	SportCodeTableTennis: SportIDTableTennis,
}

// Canonical ID Regex: SPD-[SPORT]-[6DIGIT]
var CanonicalIDRegex = regexp.MustCompile(`^SPD-(FB|IH|BB|TN|VB|RB|MM|CK|TT)-([0-9]{6})$`)

// Identity States
type IdentityState string

const (
	IdentityDiscovered IdentityState = "DISCOVERED"
	IdentityIdentified IdentityState = "IDENTIFIED"
	IdentityMapped     IdentityState = "MAPPED"
)

// Verification States
type VerificationState string

const (
	VerificationUnverified VerificationState = "UNVERIFIED"
	VerificationPending    VerificationState = "PENDING"
	VerificationVerified   VerificationState = "VERIFIED"
	VerificationFlagged    VerificationState = "FLAGGED"
	VerificationRejected   VerificationState = "REJECTED"
)

// Fixture Lifecycle States (Finished != Settled)
type FixtureLifecycleState string

const (
	FixtureNotStarted FixtureLifecycleState = "NOT_STARTED"
	FixtureLive       FixtureLifecycleState = "LIVE"
	FixtureFinished   FixtureLifecycleState = "FINISHED"
)

// Mapping Types
type MappingType string

const (
	MappingPrematch MappingType = "PREMATCH"
	MappingLive     MappingType = "LIVE"
	MappingResult   MappingType = "RESULT"
)

// FormatCanonicalFixtureID creates a standardized platform canonical fixture ID: SPD-[SPORT]-[6DIGIT].
// The sequence is an opaque, platform-generated identifier formatted with 6 digits (padded with leading zeros if < 100000).
func FormatCanonicalFixtureID(sportID int, sequence int64) (string, error) {
	code, exists := SportIDToCode[sportID]
	if !exists {
		return "", fmt.Errorf("unsupported sport ID: %d", sportID)
	}
	if sequence <= 0 {
		return "", fmt.Errorf("sequence must be positive, got %d", sequence)
	}
	// Six-digit modulo formatting if sequence exceeds 6 digits or direct padding
	seqStr := fmt.Sprintf("%06d", sequence%1000000)
	if sequence >= 100000 && sequence <= 999999 {
		seqStr = fmt.Sprintf("%06d", sequence)
	}
	return fmt.Sprintf("%s-%s-%s", Prefix, code, seqStr), nil
}

// ParseCanonicalFixtureID extracts the sport ID, sport code, and sequence number from a canonical fixture ID.
func ParseCanonicalFixtureID(canonicalID string) (int, string, int64, error) {
	matches := CanonicalIDRegex.FindStringSubmatch(strings.TrimSpace(canonicalID))
	if len(matches) != 3 {
		return 0, "", 0, fmt.Errorf("invalid canonical fixture ID format: %q (expected SPD-[SPORT]-[6DIGIT])", canonicalID)
	}
	code := matches[1]
	seqStr := matches[2]
	sportID, ok := SportCodeToID[code]
	if !ok {
		return 0, "", 0, fmt.Errorf("unknown sport code: %s", code)
	}
	seq, err := strconv.ParseInt(seqStr, 10, 64)
	if err != nil {
		return 0, "", 0, fmt.Errorf("invalid sequence in canonical ID: %w", err)
	}
	return sportID, code, seq, nil
}

// ValidateCanonicalFixtureID returns nil if canonicalID conforms to SPD-[SPORT]-[6DIGIT].
func ValidateCanonicalFixtureID(canonicalID string) error {
	if !CanonicalIDRegex.MatchString(strings.TrimSpace(canonicalID)) {
		return errors.New("INVALID_CANONICAL_FIXTURE_ID: expected format SPD-[SPORT]-[6DIGIT] (e.g. SPD-FB-456778)")
	}
	return nil
}

// NormalizeTeamName prepares a team name for natural identity hashing:
// lowercased, stripped of common punctuation and excess whitespace.
func NormalizeTeamName(team string) string {
	cleaned := strings.ToLower(strings.TrimSpace(team))
	// Replace common variations
	replacer := strings.NewReplacer(
		".", "",
		",", "",
		"-", " ",
		"_", " ",
		"/", " ",
		"'", "",
		"\"", "",
		"fc", "",
		"sc", "",
		"fk", "",
		"afc", "",
	)
	cleaned = replacer.Replace(cleaned)
	// Collapse multiple spaces
	words := strings.Fields(cleaned)
	return strings.Join(words, " ")
}

// NaturalIdentityHash computes a deterministic SHA256 hash identifying a real-world match
// based on sport, normalized teams, and exact kickoff time.
func NaturalIdentityHash(sportID int, homeTeam, awayTeam string, startTime time.Time) string {
	normHome := NormalizeTeamName(homeTeam)
	normAway := NormalizeTeamName(awayTeam)
	unixSec := startTime.UTC().Unix()

	h := sha256.New()
	fmt.Fprintf(h, "%d:%s:%s:%d", sportID, normHome, normAway, unixSec)
	return hex.EncodeToString(h.Sum(nil))
}
