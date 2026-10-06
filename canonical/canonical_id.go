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
	"RG":                 SportIDRugby, // Compatible alias
	SportCodeMMA:         SportIDMMA,
	SportCodeCricket:     SportIDCricket,
	"CR":                 SportIDCricket, // Compatible alias
	SportCodeTableTennis: SportIDTableTennis,
}

// Canonical ID Regex: SPD-[SPORT]-[6DIGIT]
var CanonicalIDRegex = regexp.MustCompile(`^SPD-(FB|IH|BB|TN|VB|RB|MM|CK|TT|CR|RG)-([0-9]{6})$`)

// Identity States
type IdentityState string

const (
	IdentityDiscovered  IdentityState = "DISCOVERED"
	IdentityIdentified  IdentityState = "IDENTIFIED"
	IdentityMapped      IdentityState = "MAPPED"
	IdentityQuarantined IdentityState = "QUARANTINED"
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
	FixturePostponed  FixtureLifecycleState = "POSTPONED"
)

// Betting States
type BettingState string

const (
	BettingOpen              BettingState = "OPEN"
	BettingBettable          BettingState = "BETTABLE"
	BettingLiveBroadcastOnly BettingState = "LIVE_BROADCAST_ONLY"
	BettingSuspended         BettingState = "SUSPENDED"
	BettingClosed            BettingState = "CLOSED"
	BettingReadOnly          BettingState = "READ_ONLY"
)

// Settlement Lifecycle States
type SettlementLifecycleState string

const (
	SettlementPending   SettlementLifecycleState = "PENDING"
	SettlementSettled   SettlementLifecycleState = "SETTLED"
	SettlementReview    SettlementLifecycleState = "REVIEW"
	SettlementCancelled SettlementLifecycleState = "CANCELLED"
)

// Mapping Types
type MappingType string

const (
	MappingPrematch  MappingType = "PREMATCH"
	MappingLive      MappingType = "LIVE"
	MappingResult    MappingType = "RESULT"
	MappingAutomatic MappingType = "AUTOMATIC"
)

// Canonical Market Codes
const (
	MarketCode1X2              = "1X2"
	MarketCodeOverUnder        = "OVER_UNDER"
	MarketCodeHandicap         = "HANDICAP"
	MarketCodeBothTeamsToScore = "BOTH_TEAMS_TO_SCORE"
	MarketCodeDoubleChance     = "DOUBLE_CHANCE"
	MarketCodeHalfTime1X2      = "HALF_TIME_1X2"
	MarketCodeDrawNoBet        = "DRAW_NO_BET"
	MarketCodeMoneyline        = "MONEYLINE"
	MarketCodeSpread           = "SPREAD"
	MarketCodeTotalPoints      = "TOTAL_POINTS"
	MarketCodeMatchWinner      = "MATCH_WINNER"
	MarketCodeSetWinner        = "SET_WINNER"
	MarketCodeTotalGames       = "TOTAL_GAMES"
	MarketCodeFightWinner      = "FIGHT_WINNER"
	MarketCodeTotalRounds      = "TOTAL_ROUNDS"
	MarketCodeMethodOfVictory  = "METHOD_OF_VICTORY"
	MarketCodeRunsOverUnder    = "RUNS_OVER_UNDER"
)

// Canonical Selection Codes
const (
	SelectionCodeHome  = "HOME"
	SelectionCodeDraw  = "DRAW"
	SelectionCodeAway  = "AWAY"
	SelectionCodeOver  = "OVER"
	SelectionCodeUnder = "UNDER"
	SelectionCodeYes   = "YES"
	SelectionCodeNo    = "NO"
	SelectionCode1X    = "1X"
	SelectionCode12    = "12"
	SelectionCodeX2    = "X2"
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
// lowercased, stripped of common punctuation, noise words, and excess whitespace.
func NormalizeTeamName(team string) string {
	cleaned := strings.ToLower(strings.TrimSpace(team))
	// Well-known synonyms & common abbreviations
	switch cleaned {
	case "psg":
		return "paris saint germain"
	case "man utd", "man united":
		return "manchester united"
	case "man city":
		return "manchester city"
	case "spurs":
		return "tottenham"
	case "wolves":
		return "wolverhampton"
	}

	// Strip common punctuation
	replacer := strings.NewReplacer(
		".", "",
		",", "",
		"-", " ",
		"_", " ",
		"/", " ",
		"'", "",
		"\"", "",
		"(", " ",
		")", " ",
	)
	cleaned = replacer.Replace(cleaned)

	words := strings.Fields(cleaned)
	noise := map[string]bool{
		"fc": true, "cf": true, "sc": true, "fk": true, "afc": true, "ac": true,
		"cd": true, "as": true, "club": true, "de": true, "the": true,
		"olympique": true,
	}

	var filtered []string
	for _, w := range words {
		if w == "psg" {
			filtered = append(filtered, "paris", "saint", "germain")
			continue
		}
		if !noise[w] {
			filtered = append(filtered, w)
		}
	}
	if len(filtered) == 0 {
		return strings.Join(words, " ")
	}
	return strings.Join(filtered, " ")
}

// NaturalCandidate encapsulates structured attributes identifying a real sporting fixture.
type NaturalCandidate struct {
	SportID           int       `json:"sport_id"`
	HomeTeam          string    `json:"home_team"`
	AwayTeam          string    `json:"away_team"`
	ScheduledStart    time.Time `json:"scheduled_start"`
	CompetitionID     string    `json:"competition_id,omitempty"`
	CompetitionName   string    `json:"competition_name,omitempty"`
	Gender            string    `json:"gender,omitempty"`        // MEN, WOMEN, UNKNOWN
	AgeCategory       string    `json:"age_category,omitempty"`  // SENIOR, U23, U21, U20, U19, U18, YOUTH, UNKNOWN
	TeamCategory      string    `json:"team_category,omitempty"` // CLUB, RESERVE, NATIONAL, UNKNOWN
	Country           string    `json:"country,omitempty"`
	Provider          string    `json:"provider,omitempty"`
	ProviderFixtureID string    `json:"provider_fixture_id,omitempty"`
}

// CleanAndClassifyCandidate resolves default categories (gender, age, team category)
// from team names, competition names, and raw tokens if not explicitly set.
func (c *NaturalCandidate) CleanAndClassifyCandidate() {
	combined := strings.ToLower(fmt.Sprintf("%s %s %s", c.HomeTeam, c.AwayTeam, c.CompetitionName))

	// Gender classification
	if c.Gender == "" || c.Gender == "UNKNOWN" {
		if strings.Contains(combined, "women") || strings.Contains(combined, "ladies") ||
			strings.Contains(combined, "(w)") || strings.Contains(combined, " w ") || strings.HasSuffix(combined, " w") {
			c.Gender = "WOMEN"
		} else {
			c.Gender = "MEN"
		}
	}

	// Age category classification
	if c.AgeCategory == "" || c.AgeCategory == "UNKNOWN" {
		if strings.Contains(combined, "u23") || strings.Contains(combined, "u-23") {
			c.AgeCategory = "U23"
		} else if strings.Contains(combined, "u21") || strings.Contains(combined, "u-21") {
			c.AgeCategory = "U21"
		} else if strings.Contains(combined, "u20") || strings.Contains(combined, "u-20") {
			c.AgeCategory = "U20"
		} else if strings.Contains(combined, "u19") || strings.Contains(combined, "u-19") {
			c.AgeCategory = "U19"
		} else if strings.Contains(combined, "u18") || strings.Contains(combined, "u-18") {
			c.AgeCategory = "U18"
		} else if strings.Contains(combined, "youth") {
			c.AgeCategory = "YOUTH"
		} else {
			c.AgeCategory = "SENIOR"
		}
	}

	// Team category classification
	if c.TeamCategory == "" || c.TeamCategory == "UNKNOWN" {
		if strings.Contains(combined, "reserves") || strings.Contains(combined, "reserve") ||
			strings.Contains(combined, "(res)") || strings.Contains(combined, " ii") || strings.Contains(combined, " 2") {
			c.TeamCategory = "RESERVE"
		} else {
			c.TeamCategory = "CLUB"
		}
	}
}

// NaturalKey computes a deterministic, collision-resistant candidate natural key.
// It incorporates sport, normalized team names, gender, age category, team category,
// and rounds kickoff to a half-hour tolerance bucket (absorbing ±15m schedule jitter across providers).
func (c NaturalCandidate) NaturalKey() string {
	candidate := c
	candidate.CleanAndClassifyCandidate()

	normHome := NormalizeTeamName(candidate.HomeTeam)
	normAway := NormalizeTeamName(candidate.AwayTeam)

	// Half-hour tolerance window: kickoffs scheduled within ±15 minutes fall in the same bucket
	roundedStart := candidate.ScheduledStart.UTC().Truncate(30 * time.Minute)
	timeBucket := roundedStart.Format("20060102-1504")

	h := sha256.New()
	fmt.Fprintf(h, "%d:%s:%s:%s:%s:%s:%s",
		candidate.SportID,
		normHome,
		normAway,
		candidate.Gender,
		candidate.AgeCategory,
		candidate.TeamCategory,
		timeBucket,
	)
	return hex.EncodeToString(h.Sum(nil))
}

// NaturalIdentityHash computes a deterministic SHA256 hash identifying a real-world match
// using NaturalCandidate resolution with kickoff tolerance and category separation.
func NaturalIdentityHash(sportID int, homeTeam, awayTeam string, startTime time.Time) string {
	c := NaturalCandidate{
		SportID:        sportID,
		HomeTeam:       homeTeam,
		AwayTeam:       awayTeam,
		ScheduledStart: startTime,
	}
	return c.NaturalKey()
}
