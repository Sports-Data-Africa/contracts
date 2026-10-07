package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Canonical Prefix
const Prefix = "SPD"

// MaxSequence is the largest sequence a 6-digit canonical ID can carry.
// FormatCanonicalFixtureID returns an error beyond this instead of silently wrapping.
const MaxSequence int64 = 999999

// KeyVersion is mixed into every natural-identity hash. Bump it whenever the
// normalisation rules change so old and new hashes can never be confused.
const KeyVersion = 2

// KickoffBucket is the width of the kickoff time bucket used by natural keys.
const KickoffBucket = 30 * time.Minute

// Sentinel errors (use errors.Is).
var (
	ErrUnknownSport       = errors.New("UNKNOWN_SPORT")
	ErrInvalidCanonicalID = errors.New("INVALID_CANONICAL_FIXTURE_ID")
	ErrSequenceOutOfRange = errors.New("SEQUENCE_OUT_OF_RANGE")
	ErrIdentityConflict   = errors.New("IDENTITY_CONFLICT")
	ErrSportBlocked       = errors.New("SPORT_BLOCKED")
	ErrUnreleasedSport    = errors.New("UNRELEASED_SPORT")
)

// Standard Sport Codes
const (
	SportCodeFootball         = "FB"
	SportCodeBasketball       = "BB"
	SportCodeTennis           = "TN"
	SportCodeIceHockey        = "IH" // Hockey
	SportCodeVolleyball       = "VB"
	SportCodeMMA              = "MM" // MMA / Boxing
	SportCodeRugby            = "RB"
	SportCodeCricket          = "CK"
	SportCodeAmericanFootball = "AF"
	SportCodeHandball         = "HB"
	SportCodeBaseball         = "BS"
	SportCodeTableTennis      = "TT"
)

// Platform-Wide Canonical Sport IDs (Independent of Provider)
const (
	SportIDFootball         = 1
	SportIDBasketball       = 2
	SportIDTennis           = 3
	SportIDIceHockey        = 4 // Hockey
	SportIDVolleyball       = 5
	SportIDMMA              = 6 // MMA / Boxing
	SportIDRugby            = 7
	SportIDCricket          = 8
	SportIDAmericanFootball = 9
	SportIDHandball         = 10
	SportIDBaseball         = 11
	// ID 12 is intentionally reserved/unused
	SportIDTableTennis = 13
)

// Mapping from numeric Sport ID to 2-letter Sport Code.
// Treat as read-only: it is shared package state and not safe for concurrent writes.
var SportIDToCode = map[int]string{
	SportIDFootball:         SportCodeFootball,
	SportIDBasketball:       SportCodeBasketball,
	SportIDTennis:           SportCodeTennis,
	SportIDIceHockey:        SportCodeIceHockey,
	SportIDVolleyball:       SportCodeVolleyball,
	SportIDMMA:              SportCodeMMA,
	SportIDRugby:            SportCodeRugby,
	SportIDCricket:          SportCodeCricket,
	SportIDAmericanFootball: SportCodeAmericanFootball,
	SportIDHandball:         SportCodeHandball,
	SportIDBaseball:         SportCodeBaseball,
	SportIDTableTennis:      SportCodeTableTennis,
}

// Mapping from 2-letter Sport Code to numeric Sport ID.
// Includes legacy aliases (RG, CR) so older callers can still be decoded;
// they are NOT valid in a canonical ID (see NormalizeCanonicalFixtureID).
var SportCodeToID = map[string]int{
	SportCodeFootball:         SportIDFootball,
	SportCodeBasketball:       SportIDBasketball,
	SportCodeTennis:           SportIDTennis,
	SportCodeIceHockey:        SportIDIceHockey,
	SportCodeVolleyball:       SportIDVolleyball,
	SportCodeMMA:              SportIDMMA,
	SportCodeRugby:            SportIDRugby,
	"RG":                      SportIDRugby, // legacy alias
	SportCodeCricket:          SportIDCricket,
	"CR":                      SportIDCricket, // legacy alias
	SportCodeAmericanFootball: SportIDAmericanFootball,
	SportCodeHandball:         SportIDHandball,
	SportCodeBaseball:         SportIDBaseball,
	SportCodeTableTennis:      SportIDTableTennis,
}

// legacyCodeAlias maps non-canonical sport codes seen in older IDs to the canonical code.
var legacyCodeAlias = map[string]string{
	"RG": SportCodeRugby,
	"CR": SportCodeCricket,
}

// CanonicalIDRegex matches ONLY canonical IDs: SPD-[SPORT]-[6DIGIT].
// Legacy codes (CR, RG) are deliberately rejected so one sport can never have two ID spellings.
var CanonicalIDRegex = regexp.MustCompile(`^SPD-(FB|BB|TN|IH|VB|MM|RB|CK|AF|HB|BS|TT)-([0-9]{6})$`)

// CanonicalSportInfo holds normalized sport metadata
type CanonicalSportInfo struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// PlatformCanonicalSports provides the master list of platform sports in canonical order
var PlatformCanonicalSports = []CanonicalSportInfo{
	{ID: SportIDFootball, Code: SportCodeFootball, Name: "Football"},
	{ID: SportIDBasketball, Code: SportCodeBasketball, Name: "Basketball"},
	{ID: SportIDTennis, Code: SportCodeTennis, Name: "Tennis"},
	{ID: SportIDIceHockey, Code: SportCodeIceHockey, Name: "Hockey"},
	{ID: SportIDVolleyball, Code: SportCodeVolleyball, Name: "Volleyball"},
	{ID: SportIDMMA, Code: SportCodeMMA, Name: "MMA"},
	{ID: SportIDRugby, Code: SportCodeRugby, Name: "Rugby"},
	{ID: SportIDCricket, Code: SportCodeCricket, Name: "Cricket"},
	{ID: SportIDAmericanFootball, Code: SportCodeAmericanFootball, Name: "American Football"},
	{ID: SportIDHandball, Code: SportCodeHandball, Name: "Handball"},
	{ID: SportIDBaseball, Code: SportCodeBaseball, Name: "Baseball"},
	{ID: SportIDTableTennis, Code: SportCodeTableTennis, Name: "Table Tennis"},
}

var sportInfoByID = func() map[int]CanonicalSportInfo {
	m := make(map[int]CanonicalSportInfo, len(PlatformCanonicalSports))
	for _, s := range PlatformCanonicalSports {
		m[s.ID] = s
	}
	return m
}()

// Sportradar sport URN number -> platform sport ID.
var sportradarSportToID = map[string]int{
	"1":   SportIDFootball,
	"2":   SportIDBasketball,
	"3":   SportIDBaseball,
	"4":   SportIDIceHockey,
	"5":   SportIDTennis,
	"6":   SportIDHandball,
	"12":  SportIDRugby,
	"16":  SportIDAmericanFootball,
	"20":  SportIDTableTennis,
	"21":  SportIDCricket,
	"23":  SportIDVolleyball,
	"117": SportIDMMA,
}

// Text slugs, codes and human names -> platform sport ID.
var sportAliases = map[string]int{
	"football": SportIDFootball, "soccer": SportIDFootball, "fb": SportIDFootball,
	"basketball": SportIDBasketball, "bb": SportIDBasketball,
	"tennis": SportIDTennis, "tn": SportIDTennis,
	"ice-hockey": SportIDIceHockey, "ice_hockey": SportIDIceHockey, "icehockey": SportIDIceHockey,
	"hockey": SportIDIceHockey, "ih": SportIDIceHockey,
	"volleyball": SportIDVolleyball, "vb": SportIDVolleyball,
	"mma": SportIDMMA, "boxing": SportIDMMA, "combat": SportIDMMA, "ufc": SportIDMMA,
	"mm": SportIDMMA, "bx": SportIDMMA,
	"rugby": SportIDRugby, "rb": SportIDRugby, "rg": SportIDRugby,
	"cricket": SportIDCricket, "ck": SportIDCricket, "cr": SportIDCricket,
	"american-football": SportIDAmericanFootball, "american_football": SportIDAmericanFootball,
	"americanfootball": SportIDAmericanFootball, "af": SportIDAmericanFootball,
	"handball": SportIDHandball, "hb": SportIDHandball,
	"baseball": SportIDBaseball, "bs": SportIDBaseball,
	"table-tennis": SportIDTableTennis, "table_tennis": SportIDTableTennis,
	"tabletennis": SportIDTableTennis, "tt": SportIDTableTennis, "ping-pong": SportIDTableTennis,
}

// NormalizeSportIDStrict maps any provider string, slug, Sportradar URN or numeric ID to the
// platform sport. Unknown or empty input returns ErrUnknownSport: it never guesses.
func NormalizeSportIDStrict(raw interface{}) (CanonicalSportInfo, error) {
	if raw == nil {
		return CanonicalSportInfo{}, fmt.Errorf("%w: nil value", ErrUnknownSport)
	}

	var str string
	switch v := raw.(type) {
	case int:
		str = strconv.Itoa(v)
	case int64:
		str = strconv.FormatInt(v, 10)
	case float64:
		str = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		str = v
	default:
		str = fmt.Sprintf("%v", v)
	}

	s := strings.ToLower(strings.TrimSpace(str))
	if s == "" {
		return CanonicalSportInfo{}, fmt.Errorf("%w: empty value", ErrUnknownSport)
	}

	// 1. Sportradar URN (e.g. "sr:sport:5" -> Tennis, "sr:sport:3" -> Baseball)
	if strings.HasPrefix(s, "sr:sport:") {
		if id, ok := sportradarSportToID[strings.TrimPrefix(s, "sr:sport:")]; ok {
			return sportInfoByID[id], nil
		}
		return CanonicalSportInfo{}, fmt.Errorf("%w: %q", ErrUnknownSport, raw)
	}

	// 2. Platform numeric ID (1..11, 13)
	if num, err := strconv.Atoi(s); err == nil {
		if info, ok := sportInfoByID[num]; ok {
			return info, nil
		}
		return CanonicalSportInfo{}, fmt.Errorf("%w: %q", ErrUnknownSport, raw)
	}

	// 3. Slugs, codes and names
	if id, ok := sportAliases[s]; ok {
		return sportInfoByID[id], nil
	}
	return CanonicalSportInfo{}, fmt.Errorf("%w: %q", ErrUnknownSport, raw)
}

// NormalizeSportID is the legacy helper. It falls back to Football for anything it cannot
// recognise, which silently mis-files unknown sports.
//
// Deprecated: use NormalizeSportIDStrict and handle the error.
func NormalizeSportID(raw interface{}) (int, string, string) {
	info, err := NormalizeSportIDStrict(raw)
	if err != nil {
		return SportIDFootball, SportCodeFootball, "Football"
	}
	return info.ID, info.Code, info.Name
}

// ----------------------------------------------------------------------------
// Canonical Sport Release Policy
// ----------------------------------------------------------------------------

var clientReleasedSports = map[int]string{
	SportIDFootball:   "Football",
	SportIDBasketball: "Basketball",
	SportIDTennis:     "Tennis",
	SportIDIceHockey:  "Hockey",
	SportIDVolleyball: "Volleyball",
	SportIDMMA:        "MMA",
	SportIDRugby:      "Rugby",
	SportIDCricket:    "Cricket",
}

var blockedSports = map[int]string{
	SportIDAmericanFootball: "American Football",
	SportIDHandball:         "Handball",
	SportIDBaseball:         "Baseball",
	SportIDTableTennis:      "Table Tennis",
}

// IsSportClientReleased returns true only for the 8 authorized client release sports.
func IsSportClientReleased(sportID int) bool {
	_, ok := clientReleasedSports[sportID]
	return ok
}

// IsSportSettleable returns true only for the 8 authorized settlement sports.
func IsSportSettleable(sportID int) bool {
	return IsSportClientReleased(sportID)
}

// IsSportBlocked returns true if the sport is explicitly blocked/quarantined from client release.
func IsSportBlocked(sportID int) bool {
	_, ok := blockedSports[sportID]
	return ok
}

// OfficialClientSportIDs returns a sorted slice of authorized client sport IDs.
func OfficialClientSportIDs() []int {
	ids := make([]int, 0, len(clientReleasedSports))
	for id := range clientReleasedSports {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// OfficialClientSportsMap returns an immutable map copy of authorized client sports.
func OfficialClientSportsMap() map[int]string {
	m := make(map[int]string, len(clientReleasedSports))
	for k, v := range clientReleasedSports {
		m[k] = v
	}
	return m
}

// BlockedSportIDs returns a sorted slice of currently blocked sport IDs.
func BlockedSportIDs() []int {
	ids := make([]int, 0, len(blockedSports))
	for id := range blockedSports {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// ----------------------------------------------------------------------------
// Canonical Fixture Alias & Cross-Provider Linkage
// ----------------------------------------------------------------------------

// FixtureAlias represents a verified mapping between a canonical fixture and an external provider event key.
type FixtureAlias struct {
	CanonicalFixtureID string    `json:"canonical_fixture_id"`
	Provider           string    `json:"provider"`
	ProviderFixtureID  string    `json:"provider_fixture_id"`
	VerifiedAt         time.Time `json:"verified_at"`
	VerificationMethod string    `json:"verification_method"`
}

// NormalizeProviderEventKey normalizes a raw external event key.
// The provider numeric event key is an external alias, not the permanent internal primary key.
// Strip sr:match: only as a provider-format normalization step.
func NormalizeProviderEventKey(provider, rawKey string) string {
	trimmed := strings.TrimSpace(rawKey)
	p := strings.ToUpper(strings.TrimSpace(provider))
	switch p {
	case "SPORTYBET":
		return strings.TrimPrefix(trimmed, "sr:match:")
	case "ODIBETS":
		return trimmed
	default:
		return strings.TrimPrefix(trimmed, "sr:match:")
	}
}

// FixtureLinkageValidation holds candidate details for verified cross-provider identity linking.
type FixtureLinkageValidation struct {
	SportID          int
	ProviderEventKey string
	ScheduledKickoff time.Time
	HomeTeam         string
	AwayTeam         string
	CompetitionName  string
}

// ValidateFixtureLinkage verifies cross-provider identity compatibility.
// Direct cross-provider identity linking is allowed only for verified compatible provider namespaces.
// The first linking operation must validate:
// - canonical sport
// - provider event key
// - scheduled kickoff (within +/- 30m window)
// - home team
// - away team
// Any identity conflict creates REVIEW and must not merge fixture records automatically.
func ValidateFixtureLinkage(a, b FixtureLinkageValidation) (bool, string) {
	if a.SportID != b.SportID {
		return false, fmt.Sprintf("%v: sport mismatch (%d vs %d)", ErrIdentityConflict, a.SportID, b.SportID)
	}
	normKeyA := strings.TrimPrefix(strings.TrimSpace(a.ProviderEventKey), "sr:match:")
	normKeyB := strings.TrimPrefix(strings.TrimSpace(b.ProviderEventKey), "sr:match:")
	if normKeyA == "" || normKeyB == "" {
		return false, fmt.Sprintf("%v: empty provider event key", ErrIdentityConflict)
	}

	sameKey := normKeyA == normKeyB

	// Check kickoff tolerance (+/- 30 min)
	timeDiff := a.ScheduledKickoff.Sub(b.ScheduledKickoff)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if !a.ScheduledKickoff.IsZero() && !b.ScheduledKickoff.IsZero() && timeDiff > 45*time.Minute {
		return false, fmt.Sprintf("%v: kickoff difference exceeds tolerance (%v)", ErrIdentityConflict, timeDiff)
	}

	// Check team names
	cleanHomeA := NormalizeTeamName(a.HomeTeam)
	cleanHomeB := NormalizeTeamName(b.HomeTeam)
	cleanAwayA := NormalizeTeamName(a.AwayTeam)
	cleanAwayB := NormalizeTeamName(b.AwayTeam)

	if cleanHomeA != "" && cleanHomeB != "" && cleanHomeA != cleanHomeB && !sameKey {
		return false, fmt.Sprintf("%v: home team mismatch (%q vs %q)", ErrIdentityConflict, a.HomeTeam, b.HomeTeam)
	}
	if cleanAwayA != "" && cleanAwayB != "" && cleanAwayA != cleanAwayB && !sameKey {
		return false, fmt.Sprintf("%v: away team mismatch (%q vs %q)", ErrIdentityConflict, a.AwayTeam, b.AwayTeam)
	}

	return true, ""
}

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
// The sequence must be in 1..MaxSequence. Out-of-range values are an error (never wrapped),
// because wrapping would silently reuse an ID that already belongs to another fixture.
func FormatCanonicalFixtureID(sportID int, sequence int64) (string, error) {
	code, exists := SportIDToCode[sportID]
	if !exists {
		return "", fmt.Errorf("%w: unsupported sport ID %d", ErrUnknownSport, sportID)
	}
	if sequence < 1 || sequence > MaxSequence {
		return "", fmt.Errorf("%w: got %d, want 1..%d", ErrSequenceOutOfRange, sequence, MaxSequence)
	}
	return fmt.Sprintf("%s-%s-%06d", Prefix, code, sequence), nil
}

// ParseCanonicalFixtureID extracts the sport ID, canonical sport code, and sequence number
// from a canonical fixture ID. Legacy codes (CR, RG) are rejected: run the input through
// NormalizeCanonicalFixtureID first if you may be holding one.
func ParseCanonicalFixtureID(canonicalID string) (int, string, int64, error) {
	matches := CanonicalIDRegex.FindStringSubmatch(strings.TrimSpace(canonicalID))
	if len(matches) != 3 {
		return 0, "", 0, fmt.Errorf("%w: %q (expected SPD-[SPORT]-[6DIGIT])", ErrInvalidCanonicalID, canonicalID)
	}
	sportID, ok := SportCodeToID[matches[1]]
	if !ok {
		return 0, "", 0, fmt.Errorf("%w: unknown sport code %s", ErrInvalidCanonicalID, matches[1])
	}
	seq, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return 0, "", 0, fmt.Errorf("%w: invalid sequence: %v", ErrInvalidCanonicalID, err)
	}
	if seq < 1 {
		return 0, "", 0, fmt.Errorf("%w: sequence must be positive", ErrInvalidCanonicalID)
	}
	return sportID, SportIDToCode[sportID], seq, nil
}

// ValidateCanonicalFixtureID returns nil if canonicalID conforms to SPD-[SPORT]-[6DIGIT].
func ValidateCanonicalFixtureID(canonicalID string) error {
	if _, _, _, err := ParseCanonicalFixtureID(canonicalID); err != nil {
		return fmt.Errorf("%w: expected format SPD-[SPORT]-[6DIGIT] (e.g. SPD-FB-456778)", ErrInvalidCanonicalID)
	}
	return nil
}

// NormalizeCanonicalFixtureID upper-cases, trims and rewrites legacy sport codes
// (CR -> CK, RG -> RB) to canonical form, then validates. Use it at system boundaries
// (APIs, event consumers) to bridge IDs produced by older services.
func NormalizeCanonicalFixtureID(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	parts := strings.Split(s, "-")
	if len(parts) == 3 && parts[0] == Prefix {
		if canonicalCode, ok := legacyCodeAlias[parts[1]]; ok {
			parts[1] = canonicalCode
			s = strings.Join(parts, "-")
		}
	}
	if err := ValidateCanonicalFixtureID(s); err != nil {
		return "", err
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// Team-name normalisation
// ---------------------------------------------------------------------------

// accentFold lower-case folds common Latin diacritics so "Atlético" == "Atletico".
var accentFold = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a", "ā", "a", "ą", "a",
	"ç", "c", "ć", "c", "č", "c",
	"ď", "d", "đ", "d", "ð", "d",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "ē", "e", "ę", "e", "ě", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ī", "i", "ı", "i",
	"ł", "l",
	"ñ", "n", "ń", "n", "ň", "n",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o", "ō", "o", "ő", "o",
	"ř", "r",
	"š", "s", "ś", "s", "ş", "s",
	"ť", "t",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ū", "u", "ů", "u", "ű", "u",
	"ý", "y", "ÿ", "y",
	"ž", "z", "ź", "z", "ż", "z",
	"ß", "ss", "æ", "ae", "œ", "oe",
)

var punctuationReplacer = strings.NewReplacer(
	".", "", ",", "", "'", "", "’", "", "`", "", "\"", "",
	"-", " ", "_", " ", "/", " ", "(", " ", ")", " ", "&", " ", "+", " ",
)

// Exact (post-tokenisation) synonyms.
var teamSynonyms = map[string]string{
	"psg":                      "paris saint germain",
	"paris sg":                 "paris saint germain",
	"man utd":                  "manchester united",
	"man united":               "manchester united",
	"manchester utd":           "manchester united",
	"man city":                 "manchester city",
	"spurs":                    "tottenham",
	"tottenham hotspur":        "tottenham",
	"wolves":                   "wolverhampton",
	"wolverhampton wanderers":  "wolverhampton",
	"inter milan":              "inter",
	"internazionale":           "inter",
	"fc internazionale milano": "inter",
}

// Tokens dropped wherever they appear.
var noiseAnywhere = map[string]bool{
	"fc": true, "cf": true, "sc": true, "fk": true, "afc": true,
	"club": true, "the": true, "de": true, "da": true, "del": true,
}

// Tokens dropped only when leading or trailing (so "AS Roma" == "Roma" but a middle "as" stays).
var noiseAtEdge = map[string]bool{
	"as": true, "ac": true, "cd": true, "olympique": true,
}

// tokenize lower-cases, folds accents, strips punctuation and splits on whitespace.
func tokenize(s string) []string {
	s = accentFold.Replace(strings.ToLower(strings.TrimSpace(s)))
	return strings.Fields(punctuationReplacer.Replace(s))
}

// NormalizeTeamName prepares a team name for matching: lower-cased, accent-folded,
// stripped of punctuation and club-type noise words, with well-known synonyms collapsed.
func NormalizeTeamName(team string) string {
	words := tokenize(team)
	if len(words) == 0 {
		return ""
	}
	joined := strings.Join(words, " ")
	if syn, ok := teamSynonyms[joined]; ok {
		return syn
	}

	filtered := make([]string, 0, len(words))
	for i, w := range words {
		if noiseAnywhere[w] {
			continue
		}
		if noiseAtEdge[w] && len(words) > 1 && (i == 0 || i == len(words)-1) {
			continue
		}
		filtered = append(filtered, w)
	}
	if len(filtered) == 0 {
		return joined
	}
	return strings.Join(filtered, " ")
}

// ---------------------------------------------------------------------------
// Candidate classification
// ---------------------------------------------------------------------------

var (
	womenRe       = regexp.MustCompile(`\b(women|womens|ladies|female|feminine|femenino|femminile|feminin)\b|\(w\)`)
	ageRe         = regexp.MustCompile(`\bu[- ]?(17|18|19|20|21|23)\b`)
	youthRe       = regexp.MustCompile(`\byouth\b`)
	reserveWordRe = regexp.MustCompile(`\breserves?\b|\(res\)`)
	ageTokenRe    = regexp.MustCompile(`\bu ?(?:17|18|19|20|21|23)\b`)
)

// Trailing tokens that mark a second/reserve side ("Barcelona B", "Ajax II").
var reserveSuffix = map[string]bool{"ii": true, "b": true, "res": true, "2": true}

// Tokens that describe the category, not the club; removed from the key name
// once the category itself has been extracted.
var qualifierTokens = map[string]bool{
	"women": true, "womens": true, "ladies": true, "female": true,
	"youth": true, "reserves": true, "reserve": true, "res": true, "ii": true,
}

func lastToken(name string) string {
	t := tokenize(name)
	if len(t) == 0 {
		return ""
	}
	return t[len(t)-1]
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
	AgeCategory       string    `json:"age_category,omitempty"`  // SENIOR, U23, U21, U20, U19, U18, U17, YOUTH, UNKNOWN
	TeamCategory      string    `json:"team_category,omitempty"` // CLUB, RESERVE, NATIONAL, UNKNOWN
	Country           string    `json:"country,omitempty"`
	Provider          string    `json:"provider,omitempty"`
	ProviderFixtureID string    `json:"provider_fixture_id,omitempty"`
}

// CleanAndClassifyCandidate fills Gender, AgeCategory and TeamCategory from team and
// competition names when they are empty or UNKNOWN. Explicit values are upper-cased and kept.
// Matching uses word boundaries, so "Group 2", "Iisalmi" or "Group B" no longer trigger
// reserve/age rules.
func (c *NaturalCandidate) CleanAndClassifyCandidate() {
	c.Gender = strings.ToUpper(strings.TrimSpace(c.Gender))
	c.AgeCategory = strings.ToUpper(strings.TrimSpace(c.AgeCategory))
	c.TeamCategory = strings.ToUpper(strings.TrimSpace(c.TeamCategory))

	combined := accentFold.Replace(strings.ToLower(c.HomeTeam + " " + c.AwayTeam + " " + c.CompetitionName))
	teamsOnly := accentFold.Replace(strings.ToLower(c.HomeTeam + " " + c.AwayTeam))

	if c.Gender == "" || c.Gender == "UNKNOWN" {
		if womenRe.MatchString(combined) || lastToken(c.HomeTeam) == "w" || lastToken(c.AwayTeam) == "w" {
			c.Gender = "WOMEN"
		} else {
			c.Gender = "MEN" // default keeps keys stable when only one provider labels the sex
		}
	}

	if c.AgeCategory == "" || c.AgeCategory == "UNKNOWN" {
		if m := ageRe.FindStringSubmatch(combined); m != nil {
			c.AgeCategory = "U" + m[1]
		} else if youthRe.MatchString(combined) {
			c.AgeCategory = "YOUTH"
		} else {
			c.AgeCategory = "SENIOR"
		}
	}

	if c.TeamCategory == "" || c.TeamCategory == "UNKNOWN" {
		// Suffix markers are checked on team names only, never on the competition
		// (a "Group B" competition must not make every side a reserve team).
		if reserveWordRe.MatchString(combined) ||
			reserveSuffix[lastToken(c.HomeTeam)] || reserveSuffix[lastToken(c.AwayTeam)] ||
			strings.Contains(teamsOnly, "(res)") {
			c.TeamCategory = "RESERVE"
		} else {
			c.TeamCategory = "CLUB"
		}
	}
}

// teamKeyName normalises a team and removes the category qualifiers that were already
// captured as Gender/Age/TeamCategory, so "Arsenal W", "Arsenal Women" and "Arsenal"
// (with Gender=WOMEN) all collapse to the same club name.
func teamKeyName(name string, c *NaturalCandidate) string {
	n := ageTokenRe.ReplaceAllString(NormalizeTeamName(name), " ")
	toks := strings.Fields(n)
	out := make([]string, 0, len(toks))
	for i, t := range toks {
		last := i == len(toks)-1
		switch {
		case qualifierTokens[t]:
			continue
		case t == "w" && last && c.Gender == "WOMEN":
			continue
		case (t == "b" || t == "2") && last && c.TeamCategory == "RESERVE":
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return NormalizeTeamName(name)
	}
	return strings.Join(out, " ")
}

func (c NaturalCandidate) keyForBucket(cl NaturalCandidate, bucket time.Time) string {
	h := sha256.New()
	fmt.Fprintf(h, "v%d|%d|%s|%s|%s|%s|%s|%s",
		KeyVersion,
		cl.SportID,
		teamKeyName(cl.HomeTeam, &cl),
		teamKeyName(cl.AwayTeam, &cl),
		cl.Gender,
		cl.AgeCategory,
		cl.TeamCategory,
		bucket.Format("20060102-1504"),
	)
	return hex.EncodeToString(h.Sum(nil))
}

// NaturalKey computes the deterministic natural key for the candidate's own kickoff bucket.
// Note: bucketing alone cannot absorb jitter across a bucket edge (19:29 vs 19:31 land in
// different buckets). When *looking up* an existing fixture use NaturalKeyCandidates instead.
func (c NaturalCandidate) NaturalKey() string {
	cl := c
	cl.CleanAndClassifyCandidate()
	return c.keyForBucket(cl, cl.ScheduledStart.UTC().Truncate(KickoffBucket))
}

// NaturalKeyCandidates returns the keys for the candidate's bucket and its two neighbours
// (own bucket first). Look up all three; if any exists, the fixture is already known.
// When creating a new fixture store NaturalKeyCandidates()[0] (== NaturalKey()).
func (c NaturalCandidate) NaturalKeyCandidates() []string {
	cl := c
	cl.CleanAndClassifyCandidate()
	b := cl.ScheduledStart.UTC().Truncate(KickoffBucket)
	return []string{
		c.keyForBucket(cl, b),
		c.keyForBucket(cl, b.Add(-KickoffBucket)),
		c.keyForBucket(cl, b.Add(KickoffBucket)),
	}
}

// NaturalIdentityHash computes the deterministic SHA-256 identifying a real-world match
// (own kickoff bucket only).
func NaturalIdentityHash(sportID int, homeTeam, awayTeam string, startTime time.Time) string {
	return NaturalCandidate{
		SportID:        sportID,
		HomeTeam:       homeTeam,
		AwayTeam:       awayTeam,
		ScheduledStart: startTime,
	}.NaturalKey()
}

// NaturalIdentityHashCandidates is the lookup variant of NaturalIdentityHash: it returns
// the hashes for the kickoff bucket and its two neighbours.
func NaturalIdentityHashCandidates(sportID int, homeTeam, awayTeam string, startTime time.Time) []string {
	return NaturalCandidate{
		SportID:        sportID,
		HomeTeam:       homeTeam,
		AwayTeam:       awayTeam,
		ScheduledStart: startTime,
	}.NaturalKeyCandidates()
}
