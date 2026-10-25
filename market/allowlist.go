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
		"1:FB_1X2_2UP":          {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_2UP", Period: PeriodFullTime, Rationale: "Early payout 2-goal lead or FT fallback supported with timeline verification"},
		"1:2UP":                 {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_2UP", Period: PeriodFullTime, Rationale: "Early payout 2-goal lead or FT fallback supported with timeline verification"},
		"1:60100":               {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_2UP", Period: PeriodFullTime, Rationale: "Early payout 2-goal lead or FT fallback supported with timeline verification"},
		"1:FB_1X2_1UP":          {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_1UP", Period: PeriodFullTime, Rationale: "Early payout 1-goal lead or FT fallback supported with timeline verification"},
		"1:1UP":                 {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_1UP", Period: PeriodFullTime, Rationale: "Early payout 1-goal lead or FT fallback supported with timeline verification"},
		"1:60200":               {SportID: 1, SportName: "Football", MarketCode: "FB_1X2_1UP", Period: PeriodFullTime, Rationale: "Early payout 1-goal lead or FT fallback supported with timeline verification"},
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
		// Football 2H markets (2H period now certified)
		"1:FB_2H_1X2":              {SportID: 1, SportName: "Football", MarketCode: "FB_2H_1X2", Period: PeriodSecondHalf, Rationale: "2nd Half score delta derived from FT minus 1H period scores"},
		"1:FB_CORNERS_1X2":         {SportID: 1, SportName: "Football", MarketCode: "FB_CORNERS_1X2", Period: PeriodFullTime, Rationale: "Corners 1X2 from DomainCorners certified feed"},
		"1:FB_CORNERS_TOTAL":        {SportID: 1, SportName: "Football", MarketCode: "FB_CORNERS_TOTAL", Period: PeriodFullTime, Rationale: "Corners O/U from DomainCorners certified feed"},
		"1:FB_CORNERS_ODD_EVEN":     {SportID: 1, SportName: "Football", MarketCode: "FB_CORNERS_ODD_EVEN", Period: PeriodFullTime, Rationale: "Corners Odd/Even from DomainCorners certified feed"},
		"1:FB_CORNERS_HANDICAP":     {SportID: 1, SportName: "Football", MarketCode: "FB_CORNERS_HANDICAP", Period: PeriodFullTime, Rationale: "Corners handicap from DomainCorners certified feed"},
		"1:FB_CARDS_TOTAL":          {SportID: 1, SportName: "Football", MarketCode: "FB_CARDS_TOTAL", Period: PeriodFullTime, Rationale: "Cards O/U from DomainCards certified feed"},
		"1:FB_CARDS_ODD_EVEN":       {SportID: 1, SportName: "Football", MarketCode: "FB_CARDS_ODD_EVEN", Period: PeriodFullTime, Rationale: "Cards Odd/Even from DomainCards certified feed"},
		"1:FB_CARDS_1X2":            {SportID: 1, SportName: "Football", MarketCode: "FB_CARDS_1X2", Period: PeriodFullTime, Rationale: "Cards 1X2 from DomainCards certified feed"},
		"1:FB_ET_1X2":               {SportID: 1, SportName: "Football", MarketCode: "FB_ET_1X2", Period: PeriodExtraTime, Rationale: "Extra time 1X2 from DomainExtraTime certified isolation"},
		"1:FB_ET_TOTAL_GOALS":        {SportID: 1, SportName: "Football", MarketCode: "FB_ET_TOTAL_GOALS", Period: PeriodExtraTime, Rationale: "Extra time totals from DomainExtraTime isolation"},
		"1:FB_PENALTY_WINNER":        {SportID: 1, SportName: "Football", MarketCode: "FB_PENALTY_WINNER", Period: PeriodPenalties, Rationale: "Penalty shootout winner from DomainPenalties isolation"},
		"1:FB_PENALTY_TOTAL":         {SportID: 1, SportName: "Football", MarketCode: "FB_PENALTY_TOTAL", Period: PeriodPenalties, Rationale: "Penalty shootout O/U from DomainPenalties isolation"},
		"1:FB_2H_BTTS":              {SportID: 1, SportName: "Football", MarketCode: "FB_2H_BTTS", Period: PeriodSecondHalf, Rationale: "2H BTTS from 2H delta score"},
		"1:FB_2H_TOTAL_GOALS":       {SportID: 1, SportName: "Football", MarketCode: "FB_TOTAL_GOALS", Period: PeriodSecondHalf, Rationale: "2H total goals from 2H delta score"},
		"1:FB_HT_CORNERS_TOTAL":     {SportID: 1, SportName: "Football", MarketCode: "FB_HT_CORNERS_TOTAL", Period: PeriodHalfTime, Rationale: "HT corners total from DomainCorners"},
		"1:FB_ANYTIME_SCORER":       {SportID: 1, SportName: "Football", MarketCode: "FB_ANYTIME_SCORER", Period: PeriodFullTime, Rationale: "Player anytime scorer from DomainPlayerStats"},
		"1:FB_FIRST_SCORER":         {SportID: 1, SportName: "Football", MarketCode: "FB_FIRST_SCORER", Period: PeriodFullTime, Rationale: "First goalscorer from DomainPlayerStats"},
		"1:FB_LAST_SCORER":          {SportID: 1, SportName: "Football", MarketCode: "FB_LAST_SCORER", Period: PeriodFullTime, Rationale: "Last goalscorer from DomainPlayerStats"},
		"1:FB_NEXT_GOAL":            {SportID: 1, SportName: "Football", MarketCode: "FB_NEXT_GOAL", Period: PeriodFullTime, Rationale: "Next goal scorer from live DomainPlayerStats"},
		"1:FB_RACE_TO_GOALS":        {SportID: 1, SportName: "Football", MarketCode: "FB_RACE_TO_GOALS", Period: PeriodFullTime, Rationale: "Race to N goals from live score"},

		// Basketball HT + Quarter markets (now certified)
		"2:BB_1H_WINNER":       {SportID: 2, SportName: "Basketball", MarketCode: "BB_1H_WINNER", Period: PeriodHalfTime, Rationale: "1H winner from Q1+Q2 period score"},
		"2:BB_1H_SPREAD":       {SportID: 2, SportName: "Basketball", MarketCode: "BB_1H_SPREAD", Period: PeriodHalfTime, Rationale: "1H spread from Q1+Q2 period score"},
		"2:BB_QUARTER_WINNER":  {SportID: 2, SportName: "Basketball", MarketCode: "BB_QUARTER_WINNER", Period: PeriodQuarter1, Rationale: "Q1 winner from period score"},
		"2:BB_QUARTER_TOTALS": {SportID: 2, SportName: "Basketball", MarketCode: "BB_QUARTER_TOTALS", Period: PeriodQuarter1, Rationale: "Q1 totals from period score"},
		"2:BB_ODD_EVEN":        {SportID: 2, SportName: "Basketball", MarketCode: "BB_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total points parity from FT score"},
		"2:BB_RACE_TO_POINTS":  {SportID: 2, SportName: "Basketball", MarketCode: "BB_RACE_TO_POINTS", Period: PeriodFullTime, Rationale: "Race to N points from live score"},
		"2:BB_TEAM_TOTALS":     {SportID: 2, SportName: "Basketball", MarketCode: "BB_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Home/away team total points from FT score"},
		"2:BB_ALT_SPREAD":      {SportID: 2, SportName: "Basketball", MarketCode: "BB_ALT_SPREAD", Period: PeriodFullTime, Rationale: "Alternative spread from FT score"},
		"2:BB_ALT_TOTAL":       {SportID: 2, SportName: "Basketball", MarketCode: "BB_ALT_TOTAL", Period: PeriodFullTime, Rationale: "Alternative total from FT score"},

		// Tennis sub-markets (all sets + extras now certified)
		"3:TN_SET_WINNER":      {SportID: 3, SportName: "Tennis", MarketCode: "TN_SET_WINNER", Period: PeriodSet1, Rationale: "Set N winner from TennisSets[].HomeGames"},
		"3:TN_TOTAL_GAMES":     {SportID: 3, SportName: "Tennis", MarketCode: "TN_TOTAL_GAMES", Period: PeriodFullTime, Rationale: "Total games O/U from DomainGames certified"},
		"3:TN_GAME_HANDICAP":   {SportID: 3, SportName: "Tennis", MarketCode: "TN_GAME_HANDICAP", Period: PeriodFullTime, Rationale: "Game handicap from DomainGames certified"},
		"3:TN_TIEBREAK_YN":     {SportID: 3, SportName: "Tennis", MarketCode: "TN_TIEBREAK_YN", Period: PeriodFullTime, Rationale: "Tiebreak yes/no from TennisSets tiebreak flag"},
		"3:TN_SET_TOTAL_GAMES": {SportID: 3, SportName: "Tennis", MarketCode: "TN_SET_TOTAL_GAMES", Period: PeriodSet1, Rationale: "Set N total games O/U from DomainGames"},
		"3:TN_SET_CORRECT_SCORE": {SportID: 3, SportName: "Tennis", MarketCode: "TN_SET_CORRECT_SCORE", Period: PeriodSet1, Rationale: "Set N correct game score from DomainGames"},

		// Ice Hockey period markets (now certified)
		"4:IH_PERIOD_1X2":    {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_PERIOD_1X2", Period: PeriodPeriod1, Rationale: "Period N winner from PeriodScores[PERIOD1]"},
		"4:IH_PERIOD_TOTALS": {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_PERIOD_TOTALS", Period: PeriodPeriod1, Rationale: "Period N totals from PeriodScores[PERIOD1]"},
		"4:IH_REGULATION_1X2": {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_REGULATION_1X2", Period: PeriodFullTime, Rationale: "Regulation 1X2 from isolated 60-min score"},
		"4:1X2":               {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_REGULATION_1X2", Period: PeriodFullTime, Rationale: "Regulation 1X2 from isolated 60-min score"},
		"4:IH_TEAM_TOTALS":    {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Team total goals from FT score"},
		"4:IH_ODD_EVEN":       {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total goals parity from FT score"},
		"4:IH_FIRST_GOAL":     {SportID: 4, SportName: "Ice Hockey", MarketCode: "IH_FIRST_GOAL", Period: PeriodFullTime, Rationale: "First team to score from live score state"},

		// Volleyball sets (now certified)
		"5:VB_SET_WINNER":       {SportID: 5, SportName: "Volleyball", MarketCode: "VB_SET_WINNER", Period: PeriodSet1, Rationale: "Set N winner from PeriodScores[SN]"},
		"5:VB_TOTAL_POINTS":     {SportID: 5, SportName: "Volleyball", MarketCode: "VB_TOTAL_POINTS", Period: PeriodFullTime, Rationale: "Total points O/U from DomainPoints certified"},
		"5:VB_POINT_HANDICAP":   {SportID: 5, SportName: "Volleyball", MarketCode: "VB_POINT_HANDICAP", Period: PeriodFullTime, Rationale: "Point handicap from DomainPoints certified"},
		"5:VB_SET_TOTAL_POINTS": {SportID: 5, SportName: "Volleyball", MarketCode: "VB_SET_TOTAL_POINTS", Period: PeriodSet1, Rationale: "Set N total points from PeriodScores[SN]"},
		"5:VB_ODD_EVEN":         {SportID: 5, SportName: "Volleyball", MarketCode: "VB_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total sets odd/even from FT score"},

		// MMA (rounds now certified)
		"6:MM_METHOD_OF_VICTORY": {SportID: 6, SportName: "MMA", MarketCode: "MM_METHOD_OF_VICTORY", Period: PeriodFullTime, Rationale: "Method of victory from DomainRounds certified"},
		"6:MM_TOTAL_ROUNDS":      {SportID: 6, SportName: "MMA", MarketCode: "MM_TOTAL_ROUNDS", Period: PeriodFullTime, Rationale: "Total rounds O/U from DomainRounds certified"},
		"6:MM_GOES_DISTANCE":     {SportID: 6, SportName: "MMA", MarketCode: "MM_GOES_DISTANCE", Period: PeriodFullTime, Rationale: "Goes distance yes/no from DomainRounds certified"},
		"6:MM_ROUND_WINNER":      {SportID: 6, SportName: "MMA", MarketCode: "MM_ROUND_WINNER", Period: PeriodRound1, Rationale: "Round N winner from DomainRounds"},
		"6:MM_FIGHT_ENDS_ROUND":  {SportID: 6, SportName: "MMA", MarketCode: "MM_FIGHT_ENDS_ROUND", Period: PeriodFullTime, Rationale: "Fight ends in round N from DomainRounds"},

		// Rugby HT (now certified)
		"7:RB_HT_1X2":        {SportID: 7, SportName: "Rugby", MarketCode: "RB_HT_1X2", Period: PeriodHalfTime, Rationale: "Half-time 1X2 from 1H PeriodScores certified"},
		"7:RB_HT_TOTAL":      {SportID: 7, SportName: "Rugby", MarketCode: "RB_HT_TOTAL_POINTS", Period: PeriodHalfTime, Rationale: "HT total points from DomainPeriodScore"},
		"7:RB_TOTAL_TRIES":   {SportID: 7, SportName: "Rugby", MarketCode: "RB_TOTAL_TRIES", Period: PeriodFullTime, Rationale: "Total tries from DomainTeamStats"},
		"7:TOTAL_TRIES":      {SportID: 7, SportName: "Rugby", MarketCode: "RB_TOTAL_TRIES", Period: PeriodFullTime, Rationale: "Total tries from DomainTeamStats"},
		"7:RB_ODD_EVEN":      {SportID: 7, SportName: "Rugby", MarketCode: "RB_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total points odd/even from FT score"},
		"7:RB_TEAM_TOTALS":   {SportID: 7, SportName: "Rugby", MarketCode: "RB_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Team total points from FT score"},
		"7:RB_DNB":           {SportID: 7, SportName: "Rugby", MarketCode: "RB_DNB", Period: PeriodFullTime, Rationale: "DNB from FT score with draw push"},
		"7:RB_DOUBLE_CHANCE": {SportID: 7, SportName: "Rugby", MarketCode: "RB_DOUBLE_CHANCE", Period: PeriodFullTime, Rationale: "Double chance from FT score"},
		"7:RB_WINNING_MARGIN": {SportID: 7, SportName: "Rugby", MarketCode: "RB_WINNING_MARGIN", Period: PeriodFullTime, Rationale: "Winning margin bands from FT score"},

		// Cricket (team stats now certified)
		"8:CK_MOST_SIXES":  {SportID: 8, SportName: "Cricket", MarketCode: "CK_MOST_SIXES", Period: PeriodFullTime, Rationale: "Most sixes from DomainTeamStats certified"},
		"8:MOST_SIXES":     {SportID: 8, SportName: "Cricket", MarketCode: "CK_MOST_SIXES", Period: PeriodFullTime, Rationale: "Most sixes from DomainTeamStats certified"},
		"8:CK_MOST_FOURS":  {SportID: 8, SportName: "Cricket", MarketCode: "CK_MOST_FOURS", Period: PeriodFullTime, Rationale: "Most fours from DomainTeamStats certified"},
		"8:CK_TEAM_RUNS":   {SportID: 8, SportName: "Cricket", MarketCode: "CK_TEAM_RUNS", Period: PeriodFullTime, Rationale: "Team runs O/U from DomainScore certified"},
		"8:CK_INNINGS_RUNS": {SportID: 8, SportName: "Cricket", MarketCode: "CK_INNINGS_RUNS", Period: PeriodInnings1, Rationale: "Innings runs O/U from DomainInnings"},
		"8:CK_TOTAL_SIXES": {SportID: 8, SportName: "Cricket", MarketCode: "CK_TOTAL_SIXES", Period: PeriodFullTime, Rationale: "Total match sixes O/U from DomainTeamStats"},
		"8:CK_TOTAL_FOURS": {SportID: 8, SportName: "Cricket", MarketCode: "CK_TOTAL_FOURS", Period: PeriodFullTime, Rationale: "Total match fours O/U from DomainTeamStats"},

		// 9. American Football (NEWLY RELEASED)
		"9:AF_MONEYLINE":      {SportID: 9, SportName: "American Football", MarketCode: "AF_MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline incl. OT from certified feed"},
		"9:AF_SPREAD":         {SportID: 9, SportName: "American Football", MarketCode: "AF_SPREAD", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game spread incl. OT from certified feed"},
		"9:AF_TOTAL_POINTS":   {SportID: 9, SportName: "American Football", MarketCode: "AF_TOTAL_POINTS", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game totals incl. OT from certified feed"},
		"9:AF_QUARTER_WINNER": {SportID: 9, SportName: "American Football", MarketCode: "AF_QUARTER_WINNER", Period: PeriodQuarter1, Rationale: "Q1 winner from PeriodScores[Q1]"},
		"9:AF_1H_WINNER":      {SportID: 9, SportName: "American Football", MarketCode: "AF_1H_WINNER", Period: PeriodHalfTime, Rationale: "1H winner from Q1+Q2 period score"},
		"9:AF_1H_SPREAD":      {SportID: 9, SportName: "American Football", MarketCode: "AF_1H_SPREAD", Period: PeriodHalfTime, Rationale: "1H spread from Q1+Q2 period score"},
		"9:AF_1H_TOTALS":      {SportID: 9, SportName: "American Football", MarketCode: "AF_1H_TOTALS", Period: PeriodHalfTime, Rationale: "1H totals from Q1+Q2 period score"},
		"9:AF_2H_WINNER":      {SportID: 9, SportName: "American Football", MarketCode: "AF_2H_WINNER", Period: PeriodSecondHalf, Rationale: "2H winner from Q3+Q4 period score"},
		"9:AF_2H_TOTALS":      {SportID: 9, SportName: "American Football", MarketCode: "AF_2H_TOTALS", Period: PeriodSecondHalf, Rationale: "2H totals from Q3+Q4 period score"},
		"9:AF_ODD_EVEN":       {SportID: 9, SportName: "American Football", MarketCode: "AF_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total points odd/even from FT score"},
		"9:AF_TEAM_TOTALS":    {SportID: 9, SportName: "American Football", MarketCode: "AF_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Team total points from FT score"},
		"9:AF_QUARTER_TOTALS": {SportID: 9, SportName: "American Football", MarketCode: "AF_QUARTER_TOTALS", Period: PeriodQuarter1, Rationale: "Q1-Q4 total points from PeriodScores"},
		"9:AF_ALT_SPREAD":     {SportID: 9, SportName: "American Football", MarketCode: "AF_ALT_SPREAD", Period: PeriodFullTime, Rationale: "Alternative spread from FT score"},
		"9:AF_ALT_TOTAL":      {SportID: 9, SportName: "American Football", MarketCode: "AF_ALT_TOTAL", Period: PeriodFullTime, Rationale: "Alternative total from FT score"},
		"9:AF_DOUBLE_RESULT":  {SportID: 9, SportName: "American Football", MarketCode: "AF_DOUBLE_RESULT", Period: PeriodFullTime, Rationale: "HT/FT double result from period + FT scores"},
		"9:AF_WINNING_MARGIN": {SportID: 9, SportName: "American Football", MarketCode: "AF_WINNING_MARGIN", Period: PeriodFullTime, Rationale: "Winning margin bands from FT score"},
		"9:AF_FIRST_SCORE":    {SportID: 9, SportName: "American Football", MarketCode: "AF_FIRST_SCORE", Period: PeriodFullTime, Rationale: "First team to score from live state"},
		"9:AF_BTTS":           {SportID: 9, SportName: "American Football", MarketCode: "AF_BTTS", Period: PeriodFullTime, Rationale: "Both teams score from FT score"},
		"9:AF_DNB":            {SportID: 9, SportName: "American Football", MarketCode: "AF_DNB", Period: PeriodFullTime, Rationale: "DNB from FT score"},

		// 10. Handball (NEWLY RELEASED)
		"10:HB_1X2":           {SportID: 10, SportName: "Handball", MarketCode: "HB_1X2", Period: PeriodFullTime, Rationale: "3-way result from FT score"},
		"10:HB_MONEYLINE":     {SportID: 10, SportName: "Handball", MarketCode: "HB_MONEYLINE", Period: PeriodFullTime, Rationale: "2-way moneyline from FT score"},
		"10:HB_HANDICAP":      {SportID: 10, SportName: "Handball", MarketCode: "HB_HANDICAP", Period: PeriodFullTime, Rationale: "Handicap from FT score"},
		"10:HB_TOTAL_GOALS":   {SportID: 10, SportName: "Handball", MarketCode: "HB_TOTAL_GOALS", Period: PeriodFullTime, Rationale: "Total goals O/U from FT score"},
		"10:HB_HT_1X2":        {SportID: 10, SportName: "Handball", MarketCode: "HB_HT_1X2", Period: PeriodHalfTime, Rationale: "HT 1X2 from DomainPeriodScore"},
		"10:HB_DNB":           {SportID: 10, SportName: "Handball", MarketCode: "HB_DNB", Period: PeriodFullTime, Rationale: "DNB from FT score"},
		"10:HB_DOUBLE_CHANCE": {SportID: 10, SportName: "Handball", MarketCode: "HB_DOUBLE_CHANCE", Period: PeriodFullTime, Rationale: "Double chance from FT score"},
		"10:HB_ODD_EVEN":      {SportID: 10, SportName: "Handball", MarketCode: "HB_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total goals parity from FT score"},
		"10:HB_HT_TOTAL_GOALS": {SportID: 10, SportName: "Handball", MarketCode: "HB_HT_TOTAL_GOALS", Period: PeriodHalfTime, Rationale: "HT total goals from DomainPeriodScore"},
		"10:HB_TEAM_TOTALS":   {SportID: 10, SportName: "Handball", MarketCode: "HB_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Team total goals from FT score"},
		"10:HB_CORRECT_SCORE": {SportID: 10, SportName: "Handball", MarketCode: "HB_CORRECT_SCORE", Period: PeriodFullTime, Rationale: "Correct score from FT score"},
		"10:HB_ASIAN_HANDICAP": {SportID: 10, SportName: "Handball", MarketCode: "HB_ASIAN_HANDICAP", Period: PeriodFullTime, Rationale: "Asian handicap from FT score"},
		"10:HB_WINNING_MARGIN": {SportID: 10, SportName: "Handball", MarketCode: "HB_WINNING_MARGIN", Period: PeriodFullTime, Rationale: "Winning margin bands from FT score"},
		"10:HB_BTTS":          {SportID: 10, SportName: "Handball", MarketCode: "HB_BTTS", Period: PeriodFullTime, Rationale: "Both teams score from FT score"},
		"10:HB_HT_DNB":        {SportID: 10, SportName: "Handball", MarketCode: "HB_HT_DNB", Period: PeriodHalfTime, Rationale: "HT DNB from DomainPeriodScore"},

		// 11. Baseball (NEWLY RELEASED)
		"11:BS_MONEYLINE":     {SportID: 11, SportName: "Baseball", MarketCode: "BS_MONEYLINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "Full game moneyline incl. extra innings"},
		"11:BS_RUN_LINE":      {SportID: 11, SportName: "Baseball", MarketCode: "BS_RUN_LINE", RequiresOT: true, Period: PeriodFullTime, Rationale: "1.5 run line from DomainInnings certified"},
		"11:BS_TOTAL_RUNS":    {SportID: 11, SportName: "Baseball", MarketCode: "BS_TOTAL_RUNS", RequiresOT: true, Period: PeriodFullTime, Rationale: "Total runs O/U from DomainInnings certified"},
		"11:BS_F5_MONEYLINE":  {SportID: 11, SportName: "Baseball", MarketCode: "BS_F5_MONEYLINE", Period: "F5", Rationale: "First 5 innings winner from innings data"},
		"11:BS_F5_TOTAL":      {SportID: 11, SportName: "Baseball", MarketCode: "BS_F5_TOTAL", Period: "F5", Rationale: "First 5 innings runs O/U from innings data"},
		"11:BS_F7_MONEYLINE":  {SportID: 11, SportName: "Baseball", MarketCode: "BS_F7_MONEYLINE", Period: "F7", Rationale: "First 7 innings winner from innings data"},
		"11:BS_F7_TOTAL":      {SportID: 11, SportName: "Baseball", MarketCode: "BS_F7_TOTAL", Period: "F7", Rationale: "First 7 innings total from innings data"},
		"11:BS_TEAM_TOTALS":   {SportID: 11, SportName: "Baseball", MarketCode: "BS_TEAM_TOTALS", Period: PeriodFullTime, Rationale: "Team total runs from DomainInnings"},
		"11:BS_ODD_EVEN":      {SportID: 11, SportName: "Baseball", MarketCode: "BS_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total runs parity from FT score"},
		"11:BS_ALT_RUN_LINE":  {SportID: 11, SportName: "Baseball", MarketCode: "BS_ALT_RUN_LINE", Period: PeriodFullTime, Rationale: "Alternative run line from FT score"},
		"11:BS_ALT_TOTAL":     {SportID: 11, SportName: "Baseball", MarketCode: "BS_ALT_TOTAL", Period: PeriodFullTime, Rationale: "Alternative total from FT score"},
		"11:BS_WINNING_MARGIN": {SportID: 11, SportName: "Baseball", MarketCode: "BS_WINNING_MARGIN", Period: PeriodFullTime, Rationale: "Winning margin bands from FT score"},
		"11:BS_BTTS":          {SportID: 11, SportName: "Baseball", MarketCode: "BS_BTTS", Period: PeriodFullTime, Rationale: "Both teams score from FT score"},
		"11:BS_FIRST_INNING":  {SportID: 11, SportName: "Baseball", MarketCode: "BS_FIRST_INNING", Period: "INNINGS1", Rationale: "Run in first inning yes/no from innings data"},
		"11:BS_DOUBLE_RESULT": {SportID: 11, SportName: "Baseball", MarketCode: "BS_DOUBLE_RESULT", Period: PeriodFullTime, Rationale: "F5/FT double result from period + FT scores"},

		// 13. Table Tennis (NEWLY RELEASED)
		"13:TT_MATCH_WINNER":  {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_MATCH_WINNER", Period: PeriodFullTime, Rationale: "Match winner from game score"},
		"13:TT_GAME_WINNER":   {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_GAME_WINNER", Period: "GAME1", Rationale: "Game N winner from PeriodScores"},
		"13:TT_TOTAL_POINTS":  {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_TOTAL_POINTS", Period: PeriodFullTime, Rationale: "Total points O/U from DomainPoints"},
		"13:TT_POINT_HANDICAP": {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_POINT_HANDICAP", Period: PeriodFullTime, Rationale: "Point handicap from DomainPoints"},
		"13:TT_GAME_HANDICAP": {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_GAME_HANDICAP", Period: PeriodFullTime, Rationale: "Game handicap from DomainGames"},
		"13:TT_CORRECT_SCORE": {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_CORRECT_SCORE", Period: PeriodFullTime, Rationale: "Correct game score from DomainGames"},
		"13:TT_ODD_EVEN":      {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_ODD_EVEN", Period: PeriodFullTime, Rationale: "Total games odd/even from FT score"},
		"13:TT_GAME_TOTALS":   {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_GAME_TOTALS", Period: "GAME1", Rationale: "Game N total points from PeriodScores"},
		"13:TT_CORRECT_GAME_SCORE": {SportID: 13, SportName: "Table Tennis", MarketCode: "TT_CORRECT_GAME_SCORE", Period: "GAME1", Rationale: "Game N correct point score from DomainPoints"},
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

	// 1. Quarantined Sports Gate: Only unreleased sports are blocked
	if !IsSportReleased(sportID) {
		return false, fmt.Sprintf("Sport ID %d (%s) is quarantined from client wagering pending full provider certification", sportID, getSportName(sportID))
	}

	extractedLine := ExtractDeterministicLine("", "", rawMarketCode+" "+marketName)

	// 2. Authoritative Allowlist Check
	allowlistMu.RLock()
	policy, inAllowlist := authoritativeAllowlist[fmt.Sprintf("%d:%s", sportID, normCode)]
	allowlistMu.RUnlock()

	var targetCode string
	p := normPeriod

	if inAllowlist {
		targetCode = policy.MarketCode
		if p == "" {
			p = policy.Period
		}
	} else {
		// Attempt provider mapping to see if it resolves to an allowlisted canonical market
		mappingResult := MapProviderMarket(ProviderMarketInput{
			Provider:      ProviderSportyBet,
			SportID:       sportID,
			RawMarketName: marketName,
			RawMarketType: rawMarketCode,
			Line:          extractedLine,
		})
		if !mappingResult.Mapped || !mappingResult.HasRule {
			// Try Odibets fallback
			mappingResult = MapProviderMarket(ProviderMarketInput{
				Provider:      ProviderOdibets,
				SportID:       sportID,
				RawMarketName: marketName,
				RawMarketType: rawMarketCode,
				Line:          extractedLine,
			})
		}
		if mappingResult.Mapped && mappingResult.HasRule {
			allowlistMu.RLock()
			policy, inAllowlist = authoritativeAllowlist[fmt.Sprintf("%d:%s", sportID, mappingResult.CanonicalMarketCode)]
			allowlistMu.RUnlock()
			if inAllowlist {
				targetCode = mappingResult.CanonicalMarketCode
				if p == "" {
					p = mappingResult.Period
				}
			}
		}
	}

	if !inAllowlist || targetCode == "" {
		return false, fmt.Sprintf("Market '%s' (Sport %d) is not in the authoritative betting allowlist", rawMarketCode, sportID)
	}

	if p == "" {
		p = PeriodFullTime
	}

	def, exists := GetMarketDefinition(sportID, targetCode, p)
	if !exists {
		mapping := MapProviderMarket(ProviderMarketInput{
			Provider:      ProviderSportyBet,
			SportID:       sportID,
			RawMarketName: marketName,
			RawMarketType: targetCode,
			Line:          extractedLine,
		})
		if mapping.Mapped && mapping.CanonicalMarketCode != "" {
			targetCode = mapping.CanonicalMarketCode
			if normPeriod == "" && mapping.Period != "" {
				p = mapping.Period
			}
			def, exists = GetMarketDefinition(sportID, targetCode, p)
		}
	}
	if !exists {
		return false, fmt.Sprintf("Market '%s' (Sport %d, Period %s) does not exist in canonical catalogue", targetCode, sportID, p)
	}
	if def.HasLine && extractedLine == nil {
		return false, fmt.Sprintf("Market '%s' requires a deterministic line parameter", targetCode)
	}

	return CanPlaceMarketCanonical(MarketPlacementRequest{
		SportID:             sportID,
		CanonicalMarketCode: targetCode,
		Period:              p,
		Line:                extractedLine,
		RuleVersion:         def.RuleVersion,
	})
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
