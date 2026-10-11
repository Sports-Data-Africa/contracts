package market

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// AllowedMarketPolicy represents an explicitly approved, settleable market.
type AllowedMarketPolicy struct {
	SportID    int    // Canonical Sport ID
	SportName  string // Human-readable sport name
	MarketCode string // Canonical market code (e.g. "FB_1X2", "OVER_UNDER")
	RequiresOT bool   // Must include overtime in settlement
	Period     string // Must match period (e.g. PeriodFullTime)
	Rationale  string // Settlement proof rationale
}

var (
	allowlistMu sync.RWMutex

	// Authoritative list of settleable markets supported post-aiscore decommissioning
	authoritativeAllowlist = map[string]AllowedMarketPolicy{
		// 1. Football (SportID 1)
		"1:1X2":                 {SportID: 1, SportName: "Football", MarketCode: "1X2", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:FB_1X2":              {SportID: 1, SportName: "Football", MarketCode: "FB_1X2", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:MATCH_RESULT":        {SportID: 1, SportName: "Football", MarketCode: "MATCH_RESULT", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:MATCH_WINNER":        {SportID: 1, SportName: "Football", MarketCode: "MATCH_WINNER", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:OVER_UNDER":          {SportID: 1, SportName: "Football", MarketCode: "OVER_UNDER", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:FB_TOTAL_GOALS":      {SportID: 1, SportName: "Football", MarketCode: "FB_TOTAL_GOALS", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:TOTAL_GOALS":         {SportID: 1, SportName: "Football", MarketCode: "TOTAL_GOALS", Period: PeriodFullTime, Rationale: "Full-time FT score confirmed by Sportradar wire"},
		"1:BTTS":                {SportID: 1, SportName: "Football", MarketCode: "BTTS", Period: PeriodFullTime, Rationale: "Both teams score >= 1 derived from FT score"},
		"1:FB_BTTS":             {SportID: 1, SportName: "Football", MarketCode: "FB_BTTS", Period: PeriodFullTime, Rationale: "Both teams score >= 1 derived from FT score"},
		"1:GG_NG":               {SportID: 1, SportName: "Football", MarketCode: "GG_NG", Period: PeriodFullTime, Rationale: "Both teams score >= 1 derived from FT score"},
		"1:BOTH_TEAMS_TO_SCORE": {SportID: 1, SportName: "Football", MarketCode: "FB_BTTS", Period: PeriodFullTime, Rationale: "Both teams score >= 1 derived from FT score"},
		"1:DOUBLE_CHANCE":       {SportID: 1, SportName: "Football", MarketCode: "DOUBLE_CHANCE", Period: PeriodFullTime, Rationale: "Derived from FT 1X2 result"},
		"1:FB_DOUBLE_CHANCE":    {SportID: 1, SportName: "Football", MarketCode: "FB_DOUBLE_CHANCE", Period: PeriodFullTime, Rationale: "Derived from FT 1X2 result"},
		"1:DRAW_NO_BET":         {SportID: 1, SportName: "Football", MarketCode: "DRAW_NO_BET", Period: PeriodFullTime, Rationale: "Derived from FT 1X2 result with push on draw"},
		"1:FB_DNB":              {SportID: 1, SportName: "Football", MarketCode: "FB_DNB", Period: PeriodFullTime, Rationale: "Derived from FT 1X2 result with push on draw"},
		"1:FB_ASIAN_HANDICAP":   {SportID: 1, SportName: "Football", MarketCode: "FB_ASIAN_HANDICAP", Period: PeriodFullTime, Rationale: "Exact integer quarter-line arithmetic supported"},
		"1:ASIAN_HANDICAP":      {SportID: 1, SportName: "Football", MarketCode: "ASIAN_HANDICAP", Period: PeriodFullTime, Rationale: "Exact integer quarter-line arithmetic supported"},

		// 2. Basketball (SportID 2) - Includes Overtime
		"2:BB_MONEYLINE":    {SportID: 2, SportName: "Basketball", MarketCode: "BB_MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game winner including OT supported by feed"},
		"2:MONEYLINE":       {SportID: 2, SportName: "Basketball", MarketCode: "MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game winner including OT supported by feed"},
		"2:WINNER":          {SportID: 2, SportName: "Basketball", MarketCode: "WINNER", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game winner including OT supported by feed"},
		"2:MATCH_WINNER":    {SportID: 2, SportName: "Basketball", MarketCode: "MATCH_WINNER", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game winner including OT supported by feed"},
		"2:BB_TOTAL_POINTS": {SportID: 2, SportName: "Basketball", MarketCode: "BB_TOTAL_POINTS", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game totals including OT supported by feed"},
		"2:OVER_UNDER":      {SportID: 2, SportName: "Basketball", MarketCode: "OVER_UNDER", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game totals including OT supported by feed"},
		"2:TOTAL_POINTS":    {SportID: 2, SportName: "Basketball", MarketCode: "TOTAL_POINTS", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game totals including OT supported by feed"},
		"2:BB_SPREAD":       {SportID: 2, SportName: "Basketball", MarketCode: "BB_SPREAD", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game spread including OT supported by feed"},
		"2:SPREAD":          {SportID: 2, SportName: "Basketball", MarketCode: "SPREAD", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game spread including OT supported by feed"},
		"2:HANDICAP":        {SportID: 2, SportName: "Basketball", MarketCode: "HANDICAP", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game spread including OT supported by feed"},

		// 4. Ice Hockey (SportID 4) - Moneyline Includes OT; Regulation 1X2 explicitly excluded!
		"4:IH_MONEYLINE": {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline including OT/SO supported"},
		"4:MONEYLINE":    {SportID: 4, SportName: "Ice Hockey", MarketCode: "MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline including OT/SO supported"},
		"4:WINNER":       {SportID: 4, SportName: "Ice Hockey", MarketCode: "WINNER", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline including OT/SO supported"},
		"4:MATCH_WINNER": {SportID: 4, SportName: "Ice Hockey", MarketCode: "MATCH_WINNER", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline including OT/SO supported"},

		// 3. Tennis (SportID 3) - Match Winner (Sets)
		"3:TN_MATCH_WINNER": {SportID: 3, SportName: "Tennis", MarketCode: "TN_MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},
		"3:MATCH_WINNER":    {SportID: 3, SportName: "Tennis", MarketCode: "MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},
		"3:WINNER":          {SportID: 3, SportName: "Tennis", MarketCode: "WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},

		// 5. Volleyball (SportID 5) - Match Winner
		"5:VB_MATCH_WINNER": {SportID: 5, SportName: "Volleyball", MarketCode: "VB_MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},
		"5:MATCH_WINNER":    {SportID: 5, SportName: "Volleyball", MarketCode: "MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},
		"5:WINNER":          {SportID: 5, SportName: "Volleyball", MarketCode: "WINNER", Period: PeriodFullTime, Rationale: "Match set score supported via SetScore wire"},

		// 7. Rugby (SportID 7) - Match 1X2 / Moneyline
		"7:RB_1X2":       {SportID: 7, SportName: "Rugby", MarketCode: "RB_1X2", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},
		"7:RB_MONEYLINE": {SportID: 7, SportName: "Rugby", MarketCode: "RB_MONEYLINE", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},
		"7:1X2":          {SportID: 7, SportName: "Rugby", MarketCode: "1X2", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},
		"7:MONEYLINE":    {SportID: 7, SportName: "Rugby", MarketCode: "MONEYLINE", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},
		"7:MATCH_WINNER": {SportID: 7, SportName: "Rugby", MarketCode: "MATCH_WINNER", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},
		"7:WINNER":       {SportID: 7, SportName: "Rugby", MarketCode: "WINNER", Period: PeriodFullTime, Rationale: "Full-time match score supported by feed"},

		// 8. Cricket (SportID 8) - Match Winner
		"8:CK_MATCH_WINNER": {SportID: 8, SportName: "Cricket", MarketCode: "CK_MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match winner supported by feed"},
		"8:MATCH_WINNER":    {SportID: 8, SportName: "Cricket", MarketCode: "MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match winner supported by feed"},
		"8:WINNER":          {SportID: 8, SportName: "Cricket", MarketCode: "WINNER", Period: PeriodFullTime, Rationale: "Match winner supported by feed"},

		// 6. MMA (SportID 6) - Fight Winner
		"6:MM_FIGHT_WINNER": {SportID: 6, SportName: "MMA", MarketCode: "MM_FIGHT_WINNER", Period: PeriodFullTime, Rationale: "Fight winner supported by feed"},
		"6:FIGHT_WINNER":    {SportID: 6, SportName: "MMA", MarketCode: "FIGHT_WINNER", Period: PeriodFullTime, Rationale: "Fight winner supported by feed"},
		"6:WINNER":          {SportID: 6, SportName: "MMA", MarketCode: "WINNER", Period: PeriodFullTime, Rationale: "Fight winner supported by feed"},
	}
)

// IsMarketSettleableAndAllowlisted checks whether a sport and market is approved for wagering.
// Returns (allowed bool, rejectionReason string).
func IsMarketSettleableAndAllowlisted(sportID int, rawMarketCode, marketName string) (bool, string) {
	// Global switch: BETTING_ALLOWLIST_ENABLED (default true)
	if strings.EqualFold(os.Getenv("BETTING_ALLOWLIST_ENABLED"), "false") {
		return true, ""
	}

	normCode := strings.ToUpper(strings.TrimSpace(rawMarketCode))
	normName := strings.ToLower(strings.TrimSpace(marketName))

	// 1. Quarantined Sports Gate: Table Tennis (SportID 13)
	if sportID == 13 || strings.Contains(normName, "table tennis") || strings.Contains(normName, "ping pong") {
		return false, "Table Tennis is quarantined from wagering until dedicated official umpire feeds are integrated"
	}

	// 2. Corners & Cards Gate (data missing post-aiscore decommissioning)
	if strings.Contains(normCode, "CORNER") || strings.Contains(normName, "corner") {
		return false, "Corners markets are suspended pending feed statistics integration"
	}
	if strings.Contains(normCode, "CARD") || strings.Contains(normName, "card") {
		return false, "Cards markets are suspended pending feed statistics integration"
	}

	// 3. Half-Time & Period/Quarter/Set/Innings Gate (missing period breakdown)
	isHT := normCode == "HT" || strings.HasPrefix(normCode, "HT_") || strings.HasSuffix(normCode, "_HT") ||
		strings.Contains(normCode, "_HT_") || strings.Contains(normCode, "HALF_TIME") || strings.Contains(normCode, "FIRST_HALF") ||
		strings.Contains(normCode, "1ST_HALF") || strings.Contains(normCode, "2ND_HALF")

	isQuarter := normCode == "Q1" || normCode == "Q2" || normCode == "Q3" || normCode == "Q4" ||
		strings.HasPrefix(normCode, "Q1_") || strings.HasPrefix(normCode, "Q2_") || strings.HasPrefix(normCode, "Q3_") || strings.HasPrefix(normCode, "Q4_") ||
		strings.HasSuffix(normCode, "_Q1") || strings.HasSuffix(normCode, "_Q2") || strings.HasSuffix(normCode, "_Q3") || strings.HasSuffix(normCode, "_Q4")

	isSet := normCode == "S1" || normCode == "S2" || normCode == "S3" ||
		strings.HasPrefix(normCode, "S1_") || strings.HasPrefix(normCode, "S2_") || strings.HasPrefix(normCode, "S3_") ||
		strings.HasSuffix(normCode, "_S1") || strings.HasSuffix(normCode, "_S2") || strings.HasSuffix(normCode, "_S3")

	isPeriod := normCode == "P1" || normCode == "P2" || normCode == "P3" ||
		strings.HasPrefix(normCode, "P1_") || strings.HasPrefix(normCode, "P2_") || strings.HasPrefix(normCode, "P3_") ||
		strings.HasSuffix(normCode, "_P1") || strings.HasSuffix(normCode, "_P2") || strings.HasSuffix(normCode, "_P3")

	isPeriodMarket := isHT || isQuarter || isSet || isPeriod ||
		strings.Contains(normName, "1st half") || strings.Contains(normName, "first half") ||
		strings.Contains(normName, "halftime") || strings.Contains(normName, "half time") || strings.Contains(normName, "2nd half") ||
		strings.Contains(normName, "quarter") || strings.Contains(normName, "1st quarter") || strings.Contains(normName, "2nd quarter") ||
		strings.Contains(normName, "set 1") || strings.Contains(normName, "set 2") || strings.Contains(normName, "1st set") ||
		strings.Contains(normName, "period 1") || strings.Contains(normName, "period 2") || strings.Contains(normName, "1st period")
	if isPeriodMarket {
		return false, "Period and interval markets (HT/Quarter/Set/Period) are suspended pending verified period feed delivery"
	}

	// 4. Ice Hockey Regulation 1X2 Gate (conflict with OT / missing period breakdowns)
	if sportID == 4 && (strings.Contains(normCode, "REGULATION") || strings.Contains(normName, "regulation") ||
		strings.Contains(normCode, "1X2") || strings.Contains(normName, "1x2") || strings.Contains(normName, "3 way")) {
		return false, "Ice Hockey 3-way regulation is suspended; only Winner/Moneyline (incl. OT) is offered"
	}

	// 5. Tennis Total Games & Handicaps
	if sportID == 3 && (strings.Contains(normName, "total games") || strings.Contains(normName, "game handicap") || strings.Contains(normName, "set handicap")) {
		return false, "Tennis games and set handicaps are suspended; only Match Winner is offered"
	}

	// 6. MMA Method & Rounds
	if sportID == 6 && (strings.Contains(normName, "method") || strings.Contains(normName, "round") || strings.Contains(normName, "distance")) {
		return false, "MMA round and method markets are suspended; only Fight Winner is offered"
	}

	// 7. Rugby Tries
	if sportID == 7 && strings.Contains(normName, "tri") {
		return false, "Rugby tries markets are suspended; only Match Winner is offered"
	}

	// 8. Cricket Props
	if sportID == 8 && (strings.Contains(normName, "six") || strings.Contains(normName, "four") || strings.Contains(normName, "batsman") || strings.Contains(normName, "run")) {
		return false, "Cricket player and innings props are suspended; only Match Winner is offered"
	}

	// Check against authoritative allowlist map
	allowlistMu.RLock()
	defer allowlistMu.RUnlock()

	lookupKey := fmt.Sprintf("%d:%s", sportID, normCode)
	if _, ok := authoritativeAllowlist[lookupKey]; ok {
		return true, ""
	}

	// Also check normalized marketName
	normNameCode := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(marketName), "/", "_"), " ", "_"))
	lookupKeyName := fmt.Sprintf("%d:%s", sportID, normNameCode)
	if _, ok := authoritativeAllowlist[lookupKeyName]; ok {
		return true, ""
	}

	// Also check raw market code alias fallback
	for k, pol := range authoritativeAllowlist {
		if pol.SportID == sportID {
			if strings.EqualFold(pol.MarketCode, normCode) || strings.EqualFold(k, fmt.Sprintf("%d:%s", sportID, normCode)) ||
				strings.EqualFold(pol.MarketCode, normNameCode) || strings.EqualFold(k, fmt.Sprintf("%d:%s", sportID, normNameCode)) {
				return true, ""
			}
		}
	}

	return false, fmt.Sprintf("Market '%s' (Sport %d) is not on the verified sportsbook settlement allowlist", rawMarketCode, sportID)
}

// NormalizeSportNameToID converts common sport names or numeric strings to canonical sport ID.
func NormalizeSportNameToID(sportName string) int {
	trimmed := strings.TrimSpace(sportName)
	switch trimmed {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "10":
		return 10
	case "11":
		return 11
	case "13":
		return 13
	}

	lower := strings.ToLower(trimmed)
	switch {
	case strings.Contains(lower, "table tennis") || strings.Contains(lower, "ping pong"):
		return 13
	case strings.Contains(lower, "football") || strings.Contains(lower, "soccer"):
		return 1
	case strings.Contains(lower, "basketball"):
		return 2
	case strings.Contains(lower, "tennis"):
		return 3
	case strings.Contains(lower, "ice hockey") || strings.Contains(lower, "hockey"):
		return 4
	case strings.Contains(lower, "volleyball"):
		return 5
	case strings.Contains(lower, "mma") || strings.Contains(lower, "ufc") || strings.Contains(lower, "boxing"):
		return 6
	case strings.Contains(lower, "rugby"):
		return 7
	case strings.Contains(lower, "cricket"):
		return 8
	case strings.Contains(lower, "american football"):
		return 9
	case strings.Contains(lower, "handball"):
		return 10
	case strings.Contains(lower, "baseball"):
		return 11
	default:
		return 1
	}
}
