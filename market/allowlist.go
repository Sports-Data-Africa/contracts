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
		"1:CORNERS":             {SportID: 1, SportName: "Football", MarketCode: "CORNERS", Period: PeriodFullTime, Rationale: "Full-time total corners supported via SportyBet/Odibets provider statistics"},
		"1:TOTAL_CORNERS":       {SportID: 1, SportName: "Football", MarketCode: "TOTAL_CORNERS", Period: PeriodFullTime, Rationale: "Full-time total corners supported via SportyBet/Odibets provider statistics"},
		"1:CORNERS_OVER_UNDER":  {SportID: 1, SportName: "Football", MarketCode: "CORNERS_OVER_UNDER", Period: PeriodFullTime, Rationale: "Full-time total corners supported via SportyBet/Odibets provider statistics"},
		"1:CORNERS_ODD_EVEN":    {SportID: 1, SportName: "Football", MarketCode: "CORNERS_ODD_EVEN", Period: PeriodFullTime, Rationale: "Corner parity supported via SportyBet/Odibets provider statistics"},
		"1:CORNERS_1X2":         {SportID: 1, SportName: "Football", MarketCode: "CORNERS_1X2", Period: PeriodFullTime, Rationale: "Corner 1X2 supported via SportyBet/Odibets provider statistics"},
		"1:CARDS":               {SportID: 1, SportName: "Football", MarketCode: "CARDS", Period: PeriodFullTime, Rationale: "Full-time total cards supported via SportyBet/Odibets provider statistics"},
		"1:TOTAL_CARDS":         {SportID: 1, SportName: "Football", MarketCode: "TOTAL_CARDS", Period: PeriodFullTime, Rationale: "Full-time total cards supported via SportyBet/Odibets provider statistics"},
		"1:CARDS_OVER_UNDER":    {SportID: 1, SportName: "Football", MarketCode: "CARDS_OVER_UNDER", Period: PeriodFullTime, Rationale: "Full-time total cards supported via SportyBet/Odibets provider statistics"},
		"1:HT_1X2":              {SportID: 1, SportName: "Football", MarketCode: "HT_1X2", Period: PeriodHalfTime, Rationale: "Half-time period score confirmed by provider live score feeds"},
		"1:1ST_HALF":            {SportID: 1, SportName: "Football", MarketCode: "1ST_HALF", Period: PeriodHalfTime, Rationale: "Half-time period score confirmed by provider live score feeds"},
		"1:FIRST_HALF":          {SportID: 1, SportName: "Football", MarketCode: "FIRST_HALF", Period: PeriodHalfTime, Rationale: "Half-time period score confirmed by provider live score feeds"},
		"1:HT_OVER_UNDER":       {SportID: 1, SportName: "Football", MarketCode: "HT_OVER_UNDER", Period: PeriodHalfTime, Rationale: "Half-time total goals confirmed by provider live score feeds"},
		"1:1ST_HALF_OVER_UNDER": {SportID: 1, SportName: "Football", MarketCode: "1ST_HALF_OVER_UNDER", Period: PeriodHalfTime, Rationale: "Half-time total goals confirmed by provider live score feeds"},
		"1:1ST_HALF_1X2":        {SportID: 1, SportName: "Football", MarketCode: "HT_1X2", Period: PeriodHalfTime, Rationale: "Half-time 1X2 confirmed by provider live score feeds"},
		"1:FIRST_HALF_1X2":      {SportID: 1, SportName: "Football", MarketCode: "HT_1X2", Period: PeriodHalfTime, Rationale: "Half-time 1X2 confirmed by provider live score feeds"},
		"1:1ST_HALF_BTTS":       {SportID: 1, SportName: "Football", MarketCode: "HT_BTTS", Period: PeriodHalfTime, Rationale: "Half-time BTTS confirmed by provider live score feeds"},
		"1:HT_BTTS":             {SportID: 1, SportName: "Football", MarketCode: "HT_BTTS", Period: PeriodHalfTime, Rationale: "Half-time BTTS confirmed by provider live score feeds"},
		"1:ODD_EVEN":            {SportID: 1, SportName: "Football", MarketCode: "ODD_EVEN", Period: PeriodFullTime, Rationale: "Match goal parity supported by FT score"},
		"1:TOTAL_ODD_EVEN":      {SportID: 1, SportName: "Football", MarketCode: "TOTAL_ODD_EVEN", Period: PeriodFullTime, Rationale: "Match goal parity supported by FT score"},

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
		// 1. Football additional verified full-time markets
		"1:EUROPEAN_HANDICAP":    {SportID: 1, SportName: "Football", MarketCode: "EUROPEAN_HANDICAP", Period: PeriodFullTime, Rationale: "3-way integer handicap confirmed by FT score"},
		"1:FB_EUROPEAN_HANDICAP": {SportID: 1, SportName: "Football", MarketCode: "FB_EUROPEAN_HANDICAP", Period: PeriodFullTime, Rationale: "3-way integer handicap confirmed by FT score"},
		"1:CORRECT_SCORE":        {SportID: 1, SportName: "Football", MarketCode: "CORRECT_SCORE", Period: PeriodFullTime, Rationale: "Exact FT score match"},
		"1:FB_CORRECT_SCORE":     {SportID: 1, SportName: "Football", MarketCode: "FB_CORRECT_SCORE", Period: PeriodFullTime, Rationale: "Exact FT score match"},
		"1:CLEAN_SHEET":          {SportID: 1, SportName: "Football", MarketCode: "CLEAN_SHEET", Period: PeriodFullTime, Rationale: "Zero opponent goals confirmed by FT score"},
		"1:FB_CLEAN_SHEET":       {SportID: 1, SportName: "Football", MarketCode: "FB_CLEAN_SHEET", Period: PeriodFullTime, Rationale: "Zero opponent goals confirmed by FT score"},
		"1:WIN_TO_NIL":           {SportID: 1, SportName: "Football", MarketCode: "WIN_TO_NIL", Period: PeriodFullTime, Rationale: "Win and clean sheet confirmed by FT score"},
		"1:FB_WIN_TO_NIL":        {SportID: 1, SportName: "Football", MarketCode: "FB_WIN_TO_NIL", Period: PeriodFullTime, Rationale: "Win and clean sheet confirmed by FT score"},

		// Phase 1 Additional verified Football markets
		"1:FB_HT_BTTS":            {SportID: 1, SportName: "Football", MarketCode: "FB_HT_BTTS", Period: PeriodHalfTime, Rationale: "1st Half BTTS confirmed by 1H score"},
		"1:FB_HT_DOUBLE_CHANCE":   {SportID: 1, SportName: "Football", MarketCode: "FB_HT_DOUBLE_CHANCE", Period: PeriodHalfTime, Rationale: "1st Half Double Chance confirmed by 1H score"},
		"1:FB_HT_DNB":             {SportID: 1, SportName: "Football", MarketCode: "FB_HT_DNB", Period: PeriodHalfTime, Rationale: "1st Half DNB confirmed by 1H score"},
		"1:FB_HT_ASIAN_HANDICAP":  {SportID: 1, SportName: "Football", MarketCode: "FB_HT_ASIAN_HANDICAP", Period: PeriodHalfTime, Rationale: "1st Half Asian Handicap confirmed by 1H score"},
		"1:FB_HT_EUROPEAN_HCP":    {SportID: 1, SportName: "Football", MarketCode: "FB_HT_EUROPEAN_HCP", Period: PeriodHalfTime, Rationale: "1st Half European Handicap confirmed by 1H score"},
		"1:FB_HT_CORRECT_SCORE":   {SportID: 1, SportName: "Football", MarketCode: "FB_HT_CORRECT_SCORE", Period: PeriodHalfTime, Rationale: "1st Half Correct Score confirmed by 1H score"},
		"1:FB_HT_CLEAN_SHEET":     {SportID: 1, SportName: "Football", MarketCode: "FB_HT_CLEAN_SHEET", Period: PeriodHalfTime, Rationale: "1st Half Clean Sheet confirmed by 1H score"},
		"1:FB_HT_WIN_TO_NIL":      {SportID: 1, SportName: "Football", MarketCode: "FB_HT_WIN_TO_NIL", Period: PeriodHalfTime, Rationale: "1st Half Win to Nil confirmed by 1H score"},
		"1:FB_HOME_TOTALS_HT":     {SportID: 1, SportName: "Football", MarketCode: "FB_HOME_TOTALS_HT", Period: PeriodHalfTime, Rationale: "1st Half Home Goals confirmed by 1H score"},
		"1:FB_AWAY_TOTALS_HT":     {SportID: 1, SportName: "Football", MarketCode: "FB_AWAY_TOTALS_HT", Period: PeriodHalfTime, Rationale: "1st Half Away Goals confirmed by 1H score"},
		"1:FB_EXACT_GOALS":        {SportID: 1, SportName: "Football", MarketCode: "FB_EXACT_GOALS", Period: PeriodFullTime, Rationale: "Exact goals confirmed by FT score"},
		"1:FB_EXACT_GOALS_HT":     {SportID: 1, SportName: "Football", MarketCode: "FB_EXACT_GOALS_HT", Period: PeriodHalfTime, Rationale: "1st Half Exact goals confirmed by 1H score"},
		"1:FB_GOAL_RANGE":         {SportID: 1, SportName: "Football", MarketCode: "FB_GOAL_RANGE", Period: PeriodFullTime, Rationale: "Goal range confirmed by FT score"},
		"1:FB_GOAL_RANGE_HT":      {SportID: 1, SportName: "Football", MarketCode: "FB_GOAL_RANGE_HT", Period: PeriodHalfTime, Rationale: "1st Half Goal range confirmed by 1H score"},
		"1:FB_HOME_EXACT_GOALS":   {SportID: 1, SportName: "Football", MarketCode: "FB_HOME_EXACT_GOALS", Period: PeriodFullTime, Rationale: "Home exact goals confirmed by FT score"},
		"1:FB_AWAY_EXACT_GOALS":   {SportID: 1, SportName: "Football", MarketCode: "FB_AWAY_EXACT_GOALS", Period: PeriodFullTime, Rationale: "Away exact goals confirmed by FT score"},
		"1:FB_HOME_ODD_EVEN":      {SportID: 1, SportName: "Football", MarketCode: "FB_HOME_ODD_EVEN", Period: PeriodFullTime, Rationale: "Home goal parity confirmed by FT score"},
		"1:FB_AWAY_ODD_EVEN":      {SportID: 1, SportName: "Football", MarketCode: "FB_AWAY_ODD_EVEN", Period: PeriodFullTime, Rationale: "Away goal parity confirmed by FT score"},
		"1:FB_HT_ODD_EVEN":        {SportID: 1, SportName: "Football", MarketCode: "FB_HT_ODD_EVEN", Period: PeriodHalfTime, Rationale: "1st Half goal parity confirmed by 1H score"},
		"1:FB_HIGHEST_SCORING_HALF": {SportID: 1, SportName: "Football", MarketCode: "FB_HIGHEST_SCORING_HALF", Period: PeriodFullTime, Rationale: "Highest scoring half confirmed by FT and 1H scores"},
		"1:FB_HOME_HIGHEST_HALF":  {SportID: 1, SportName: "Football", MarketCode: "FB_HOME_HIGHEST_HALF", Period: PeriodFullTime, Rationale: "Home highest half confirmed by FT and 1H scores"},
		"1:FB_AWAY_HIGHEST_HALF":  {SportID: 1, SportName: "Football", MarketCode: "FB_AWAY_HIGHEST_HALF", Period: PeriodFullTime, Rationale: "Away highest half confirmed by FT and 1H scores"},
		"1:FB_WIN_BOTH_HALVES":    {SportID: 1, SportName: "Football", MarketCode: "FB_WIN_BOTH_HALVES", Period: PeriodFullTime, Rationale: "Win both halves confirmed by FT and 1H scores"},
		"1:FB_SCORE_BOTH_HALVES":  {SportID: 1, SportName: "Football", MarketCode: "FB_SCORE_BOTH_HALVES", Period: PeriodFullTime, Rationale: "Score in both halves confirmed by FT and 1H scores"},
		"1:FB_TO_WIN_EITHER_HALF": {SportID: 1, SportName: "Football", MarketCode: "FB_TO_WIN_EITHER_HALF", Period: PeriodFullTime, Rationale: "Win either half confirmed by FT and 1H scores"},
		"1:FB_MULTI_GOALS":        {SportID: 1, SportName: "Football", MarketCode: "FB_MULTI_GOALS", Period: PeriodFullTime, Rationale: "Multi goals confirmed by FT score"},
		"1:FB_RESULT_BTTS":        {SportID: 1, SportName: "Football", MarketCode: "FB_RESULT_BTTS", Period: PeriodFullTime, Rationale: "Match result and BTTS confirmed by FT score"},
		"1:FB_RESULT_TOTALS":      {SportID: 1, SportName: "Football", MarketCode: "FB_RESULT_TOTALS", Period: PeriodFullTime, Rationale: "Match result and Over/Under confirmed by FT score"},
		"1:FB_BTTS_TOTALS":        {SportID: 1, SportName: "Football", MarketCode: "FB_BTTS_TOTALS", Period: PeriodFullTime, Rationale: "BTTS and Over/Under confirmed by FT score"},
	}
)

// CanPlaceMarket verifies whether a market can be accepted for placement on a fixture.
// It enforces canonical registry presence, sport release rules, and verified result contracts.
// Unknown or unmapped markets fail closed.
func CanPlaceMarket(sportID int, rawMarketCode, marketName, period string) (bool, string) {
	return evaluateMarketPlacement(sportID, rawMarketCode, marketName, period)
}

// IsMarketSettleableAndAllowlisted checks whether a sport and market is approved for wagering.
// Returns (allowed bool, rejectionReason string).
func IsMarketSettleableAndAllowlisted(sportID int, rawMarketCode, marketName string) (bool, string) {
	return evaluateMarketPlacement(sportID, rawMarketCode, marketName, "")
}

func evaluateMarketPlacement(sportID int, rawMarketCode, marketName, period string) (bool, string) {
	// Global switch: BETTING_ALLOWLIST_ENABLED can ONLY disable checks in non-production test environments
	if strings.EqualFold(os.Getenv("BETTING_ALLOWLIST_ENABLED"), "false") {
		if !strings.EqualFold(os.Getenv("ENVIRONMENT"), "production") && !strings.EqualFold(os.Getenv("GO_ENV"), "production") {
			return true, ""
		}
	}

	normCode := strings.ToUpper(strings.TrimSpace(rawMarketCode))
	normPeriod := strings.ToUpper(strings.TrimSpace(period))

	// 1. Quarantined Sports Gate: Table Tennis (SportID 13) and non-released sports (9, 10, 11)
	if sportID == 13 || strings.Contains(strings.ToLower(marketName), "table tennis") || strings.Contains(strings.ToLower(marketName), "ping pong") {
		return false, "Table Tennis is quarantined from wagering until dedicated official umpire feeds are integrated"
	}
	if sportID == 9 || sportID == 10 || sportID == 11 {
		return false, fmt.Sprintf("Sport ID %d is quarantined from client wagering pending full provider certification", sportID)
	}

	// 2. Direct Canonical Capability Registry Check if normCode is a known canonical code
	extractedLine := ExtractDeterministicLine("", "", rawMarketCode+" "+marketName)
	if normCode != "" {
		p := normPeriod
		if p == "" {
			p = PeriodFullTime
		}
		if def, exists := GetMarketDefinition(sportID, normCode, p); exists {
			if def.HasLine && extractedLine == nil {
				return false, fmt.Sprintf("Market '%s' requires a deterministic line parameter", normCode)
			}
			allowed, reason := CanPlaceMarketCanonical(MarketPlacementRequest{
				SportID:             sportID,
				CanonicalMarketCode: normCode,
				Period:              p,
				Line:                extractedLine,
				RuleVersion:         def.RuleVersion,
			})
			return allowed, reason
		}
	}

	// 3. Provider Mapping Entry Point: Normalize raw/provider market code & name to canonical definition
	mappingResult := MapProviderMarket(ProviderMarketInput{
		Provider:      ProviderSportyBet,
		SportID:       sportID,
		RawMarketName: marketName,
		RawMarketType: rawMarketCode,
		Line:          extractedLine,
	})

	if mappingResult.Mapped && mappingResult.HasRule {
		p := normPeriod
		if p == "" {
			p = mappingResult.Period
		}
		allowed, reason := CanPlaceMarketCanonical(MarketPlacementRequest{
			SportID:             sportID,
			CanonicalMarketCode: mappingResult.CanonicalMarketCode,
			Period:              p,
			Line:                mappingResult.Line,
			RuleVersion:         mappingResult.RuleVersion,
		})
		if !allowed {
			return false, reason
		}
		if mappingResult.Status == MarketStatusActive {
			return true, ""
		}
		return false, mappingResult.Reason
	}

	// Try with Odibets provider mapping as secondary fallback
	odiResult := MapProviderMarket(ProviderMarketInput{
		Provider:      ProviderOdibets,
		SportID:       sportID,
		RawMarketName: marketName,
		RawMarketType: rawMarketCode,
		Line:          extractedLine,
	})
	if odiResult.Mapped && odiResult.HasRule {
		p := normPeriod
		if p == "" {
			p = odiResult.Period
		}
		allowed, reason := CanPlaceMarketCanonical(MarketPlacementRequest{
			SportID:             sportID,
			CanonicalMarketCode: odiResult.CanonicalMarketCode,
			Period:              p,
			Line:                odiResult.Line,
			RuleVersion:         odiResult.RuleVersion,
		})
		if !allowed {
			return false, reason
		}
		if odiResult.Status == MarketStatusActive {
			return true, ""
		}
		return false, odiResult.Reason
	}

	// Strictly FAIL CLOSED on unknown, unmapped, or unverified markets.
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
	case strings.Contains(lower, "american football") || strings.Contains(lower, "nfl"):
		return 9
	case strings.Contains(lower, "table tennis") || strings.Contains(lower, "ping pong"):
		return 13
	case strings.Contains(lower, "football") || strings.Contains(lower, "soccer"):
		return 1
	case strings.Contains(lower, "basketball") || strings.Contains(lower, "nba"):
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
	case strings.Contains(lower, "handball"):
		return 10
	case strings.Contains(lower, "baseball") || strings.Contains(lower, "mlb"):
		return 11
	default:
		return 0
	}
}
