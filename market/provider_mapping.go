package market

import (
	"fmt"
	"strconv"
	"strings"
)

// ProviderMarketInput represents incoming market data from a provider (SportyBet or Odibets).
type ProviderMarketInput struct {
	Provider       string   `json:"provider"`
	SportID        int      `json:"sport_id"`
	RawMarketID    string   `json:"raw_market_id"`
	RawMarketName  string   `json:"raw_market_name"`
	RawMarketType  string   `json:"raw_market_type"`
	RawSpecifier   string   `json:"raw_specifier,omitempty"`
	RawOutcomeID   string   `json:"raw_outcome_id"`
	RawOutcomeName string   `json:"raw_outcome_name"`
	RawPayloadHash string   `json:"raw_payload_hash,omitempty"`
	Line           *float64 `json:"line,omitempty"`
}

// CanonicalMarketMappingResult represents the verified canonical mapping for customer offering.
type CanonicalMarketMappingResult struct {
	Mapped              bool     `json:"mapped"`
	HasRule             bool     `json:"has_rule"`
	Provider            string   `json:"provider"`
	ProviderMarketID    string   `json:"provider_market_id"`
	ProviderOutcomeID   string   `json:"provider_outcome_id"`
	RawPayloadHash      string   `json:"raw_payload_hash,omitempty"`
	SportID             int      `json:"sport_id"`
	CanonicalMarketCode string   `json:"canonical_market_code"`
	CanonicalMarketName string   `json:"canonical_market_name"`
	FormulaType         string   `json:"formula_type"` // 1X2, MONEYLINE, TOTALS, HANDICAP, BTTS, DOUBLE_CHANCE, DNB, ODD_EVEN, EXACT
	Metric              string   `json:"metric"`       // goals, points, corners, cards, sets, games, rounds, etc.
	Period              string   `json:"period"`       // FT, HT, 2H, Q1-Q4, SET1-SET5, PERIOD1-PERIOD3
	Target              string   `json:"target"`       // match, home, away
	NormalizedSelection string   `json:"normalized_selection"`
	Line                *float64 `json:"line,omitempty"`
	LineQuarters        *int     `json:"line_quarters,omitempty"`
	RequiredDomain      string   `json:"required_domain"`
	SettlementTiming    string   `json:"settlement_timing"`
	RuleVersion         int      `json:"rule_version"`
	Status              string   `json:"status"` // ACTIVE, SUSPENDED, QUARANTINED, DISABLED_UNSUPPORTED
	Reason              string   `json:"reason,omitempty"`
}

// ExtractDeterministicLine attempts to extract an exact numerical line from specifiers, outcome name, or market description.
// Returns nil if no deterministic line is present.
func ExtractDeterministicLine(specifier, outcomeName, marketDesc string) *float64 {
	// 1. Try specifier (e.g. "total=2.5", "hcp=-1.5", "goal=0.5")
	if specifier != "" {
		for _, part := range strings.Split(specifier, "&") {
			kv := strings.Split(part, "=")
			if len(kv) == 2 {
				k := strings.ToLower(strings.TrimSpace(kv[0]))
				if k == "total" || k == "hcp" || k == "handicap" || k == "goal" || k == "points" || k == "line" {
					if parsed, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64); err == nil {
						return &parsed
					}
				}
			}
		}
	}

	splitTokens := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '_' || r == '/'
		})
	}

	// 2. Try parsing from outcome text (e.g. "Over 2.5", "Under 0.5", "+1.5", "Home (-1.5)")
	if outcomeName != "" {
		for _, token := range splitTokens(outcomeName) {
			clean := strings.Trim(token, "(),")
			clean = strings.TrimPrefix(clean, "+")
			if parsed, err := strconv.ParseFloat(clean, 64); err == nil {
				return &parsed
			}
		}
	}

	// 3. Try parsing from market description (e.g. "Over/Under 2.5 Goals", "Handicap 0:1", "OVER_UNDER_2.5")
	if marketDesc != "" {
		for _, token := range splitTokens(marketDesc) {
			clean := strings.Trim(token, "(),")
			clean = strings.TrimPrefix(clean, "+")
			if parsed, err := strconv.ParseFloat(clean, 64); err == nil {
				return &parsed
			}
		}
	}

	return nil
}

// MapProviderMarket normalizes incoming SportyBet or Odibets market offerings into canonical definitions.
// Zero Unsupported Markets Invariant: If unmapped or no settlement rule exists -> DISABLED_UNSUPPORTED.
func MapProviderMarket(input ProviderMarketInput) CanonicalMarketMappingResult {
	provider := strings.ToUpper(strings.TrimSpace(input.Provider))
	sportID := input.SportID

	var canonicalCode string
	var period = PeriodFullTime
	var normSel string
	var line = input.Line
	var lineQuarters *int

	// Extract deterministic line if not explicitly supplied
	if line == nil {
		line = ExtractDeterministicLine(input.RawSpecifier, input.RawOutcomeName, input.RawMarketName)
	}

	rawID := strings.TrimSpace(input.RawMarketID)
	if rawID == "" {
		rawID = strings.TrimSpace(input.RawMarketType)
	}
	descRaw := strings.ToLower(strings.TrimSpace(input.RawMarketName + " " + input.RawMarketType))
	normDesc := strings.ReplaceAll(descRaw, "_", " ")
	rawSel := strings.TrimSpace(input.RawOutcomeName)

	switch provider {
	case ProviderSportyBet:
		canonicalCode, period, normSel = mapSportyBet(sportID, rawID, normDesc, rawSel)
	case ProviderOdibets:
		canonicalCode, period, normSel = mapOdibets(sportID, rawID, normDesc, rawSel)
	default:
		return CanonicalMarketMappingResult{
			Provider:            provider,
			ProviderMarketID:    input.RawMarketID,
			ProviderOutcomeID:   input.RawOutcomeID,
			RawPayloadHash:      input.RawPayloadHash,
			SportID:             sportID,
			Status:              MarketStatusDisabledUnsupported,
			Mapped:              false,
			HasRule:             false,
			Reason:              fmt.Sprintf("UNKNOWN_PROVIDER: %s", input.Provider),
		}
	}

	if canonicalCode == "" {
		return CanonicalMarketMappingResult{
			Provider:            provider,
			ProviderMarketID:    input.RawMarketID,
			ProviderOutcomeID:   input.RawOutcomeID,
			RawPayloadHash:      input.RawPayloadHash,
			SportID:             sportID,
			Status:              MarketStatusDisabledUnsupported,
			Mapped:              false,
			HasRule:             false,
			Reason:              fmt.Sprintf("UNMAPPED_MARKET: provider=%s, sport=%d, id=%s, desc=%s", provider, sportID, input.RawMarketID, normDesc),
		}
	}

	// Check canonical catalogue definition
	def, exists := GetMarketDefinition(sportID, canonicalCode, period)
	if !exists || def.Status != MarketStatusActive {
		return CanonicalMarketMappingResult{
			Provider:            provider,
			ProviderMarketID:    input.RawMarketID,
			ProviderOutcomeID:   input.RawOutcomeID,
			RawPayloadHash:      input.RawPayloadHash,
			SportID:             sportID,
			Status:              MarketStatusDisabledUnsupported,
			Mapped:              false,
			HasRule:             false,
			CanonicalMarketCode: canonicalCode,
			Period:              period,
			Reason:              fmt.Sprintf("NO_ACTIVE_CATALOGUE_DEF for code=%s, period=%s", canonicalCode, period),
		}
	}

	// Verify settlement rule exists in registry
	rule, hasRule := Get(sportID, canonicalCode, period, def.RuleVersion)
	if !hasRule {
		return CanonicalMarketMappingResult{
			Provider:            provider,
			ProviderMarketID:    input.RawMarketID,
			ProviderOutcomeID:   input.RawOutcomeID,
			RawPayloadHash:      input.RawPayloadHash,
			SportID:             sportID,
			Status:              MarketStatusDisabledUnsupported,
			Mapped:              true,
			HasRule:             false,
			CanonicalMarketCode: canonicalCode,
			Period:              period,
			Reason:              fmt.Sprintf("NO_SETTLEMENT_RULE_REGISTERED for code=%s, period=%s, v=%d", canonicalCode, period, def.RuleVersion),
		}
	}

	// Deterministic Line Requirement: if catalogue definition requires a line, line must be non-nil
	if def.HasLine && line == nil {
		return CanonicalMarketMappingResult{
			Provider:            provider,
			ProviderMarketID:    input.RawMarketID,
			ProviderOutcomeID:   input.RawOutcomeID,
			RawPayloadHash:      input.RawPayloadHash,
			SportID:             sportID,
			Status:              MarketStatusSuspended,
			Mapped:              true,
			HasRule:             true,
			CanonicalMarketCode: canonicalCode,
			CanonicalMarketName: def.MarketName,
			Period:              period,
			NormalizedSelection: normSel,
			RequiredDomain:      rule.RequiredDomain(),
			SettlementTiming:    rule.Timing(),
			RuleVersion:         def.RuleVersion,
			Reason:              "MISSING_DETERMINISTIC_LINE: market requires exact numeric line",
		}
	}

	// Calculate LineQuarters for handicap/totals
	if line != nil {
		q := FloatToLineQuarters(*line)
		lineQuarters = &q
	}

	formula := def.FormulaType
	metric := def.Metric
	target := def.Target
	if formula == "" {
		formula, metric, target = inferFormulaMetricTarget(canonicalCode)
	}

	// Evaluate Capability Registry (Released Sport, Certified Domain, Supported Period)
	allowed, reason := CanPlaceMarketCanonical(MarketPlacementRequest{
		SportID:             sportID,
		CanonicalMarketCode: canonicalCode,
		Period:              period,
		Line:                line,
		RuleVersion:         def.RuleVersion,
	})

	status := MarketStatusActive
	if !allowed {
		// Classify status based on reason
		if strings.Contains(reason, "quarantined") {
			status = MarketStatusQuarantined
		} else if strings.Contains(reason, "suspended") {
			status = MarketStatusSuspended
		} else {
			status = MarketStatusDisabledUnsupported
		}
	}

	return CanonicalMarketMappingResult{
		Mapped:              true,
		HasRule:             true,
		Provider:            provider,
		ProviderMarketID:    input.RawMarketID,
		ProviderOutcomeID:   input.RawOutcomeID,
		RawPayloadHash:      input.RawPayloadHash,
		SportID:             sportID,
		CanonicalMarketCode: canonicalCode,
		CanonicalMarketName: def.MarketName,
		FormulaType:         formula,
		Metric:              metric,
		Period:              period,
		Target:              target,
		NormalizedSelection: normSel,
		Line:                line,
		LineQuarters:        lineQuarters,
		RequiredDomain:      rule.RequiredDomain(),
		SettlementTiming:    rule.Timing(),
		RuleVersion:         def.RuleVersion,
		Status:              status,
		Reason:              reason,
	}
}

func mapSportyBet(sportID int, rawID, normDesc, rawSel string) (string, string, string) {
	normSel := normalizeSelection(rawSel)

	switch sportID {
	case 1: // Football
		switch {
		case rawID == "1" || rawID == "FB_1X2" || strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "3 way result") || strings.Contains(normDesc, "match result"):
			if strings.Contains(normDesc, "1st half") || strings.Contains(normDesc, "1st_half") || strings.Contains(normDesc, "first half") || strings.Contains(normDesc, "first_half") || strings.Contains(normDesc, "ht") {
				return "FB_HT_1X2", PeriodHalfTime, normSel
			}
			if strings.Contains(normDesc, "2nd half") || strings.Contains(normDesc, "2nd_half") || strings.Contains(normDesc, "second half") || strings.Contains(normDesc, "second_half") || strings.Contains(normDesc, "2h") {
				return "FB_2H_1X2", PeriodSecondHalf, normSel
			}
			return "FB_1X2", PeriodFullTime, normSel
		case (rawID == "18" || rawID == "FB_TOTAL_GOALS" || strings.Contains(normDesc, "over/under") || strings.Contains(normDesc, "over_under") || strings.Contains(normDesc, "over under") || strings.Contains(normDesc, "total goals") || strings.Contains(normDesc, "total_goals") || strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "total")) &&
			!strings.Contains(normDesc, "corner") && !strings.Contains(normDesc, "card") && !strings.Contains(normDesc, "throw") &&
			!strings.Contains(normDesc, "offside") && !strings.Contains(normDesc, "foul") && !strings.Contains(normDesc, "booking"):
			if strings.Contains(normDesc, "1st half") || strings.Contains(normDesc, "1st_half") || strings.Contains(normDesc, "first half") || strings.Contains(normDesc, "first_half") || strings.Contains(normDesc, "ht") {
				return "FB_TOTAL_GOALS", PeriodHalfTime, normalizeOverUnder(rawSel)
			}
			if strings.Contains(normDesc, "2nd half") || strings.Contains(normDesc, "2nd_half") || strings.Contains(normDesc, "second half") || strings.Contains(normDesc, "second_half") || strings.Contains(normDesc, "2h") {
				return "FB_TOTAL_GOALS", PeriodSecondHalf, normalizeOverUnder(rawSel)
			}
			return "FB_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		case rawID == "29" || strings.Contains(normDesc, "both teams to score") || strings.Contains(normDesc, "gg/ng") || strings.Contains(normDesc, "btts"):
			return "FB_BTTS", PeriodFullTime, normalizeYesNo(rawSel)
		case rawID == "10" || strings.Contains(normDesc, "double chance"):
			return "FB_DOUBLE_CHANCE", PeriodFullTime, normalizeDoubleChance(rawSel)
		case rawID == "60" || strings.Contains(normDesc, "draw no bet") || strings.Contains(normDesc, "dnb"):
			return "FB_DNB", PeriodFullTime, normSel
		case rawID == "16" || strings.Contains(normDesc, "asian handicap"):
			return "FB_ASIAN_HANDICAP", PeriodFullTime, normSel
		case (rawID == "14" || strings.Contains(normDesc, "handicap") || strings.Contains(normDesc, "european handicap")) &&
			!strings.Contains(normDesc, "asian") && !strings.Contains(normDesc, "corner") && !strings.Contains(normDesc, "card") &&
			!strings.Contains(normDesc, "prop"):
			return "FB_EUROPEAN_HANDICAP", PeriodFullTime, normSel
		case rawID == "7" || strings.Contains(normDesc, "correct score"):
			return "FB_CORRECT_SCORE", PeriodFullTime, rawSel
		case strings.Contains(normDesc, "ht/ft") || strings.Contains(normDesc, "half time/full time"):
			return "FB_HT_FT", PeriodFullTime, rawSel
		case rawID == "20" || strings.Contains(normDesc, "odd/even") || strings.Contains(normDesc, "odd or even"):
			return "FB_ODD_EVEN", PeriodFullTime, normalizeOddEven(rawSel)
		case strings.Contains(normDesc, "clean sheet"):
			return "FB_CLEAN_SHEET", PeriodFullTime, normSel
		case strings.Contains(normDesc, "win to nil"):
			return "FB_WIN_TO_NIL", PeriodFullTime, normSel
		case strings.Contains(normDesc, "home total") || strings.Contains(normDesc, "team 1 total"):
			return "FB_HOME_TOTALS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "away total") || strings.Contains(normDesc, "team 2 total"):
			return "FB_AWAY_TOTALS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "corner") && (strings.Contains(normDesc, "over/under") || strings.Contains(normDesc, "total")):
			return "FB_CORNERS_TOTAL", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "corner") && strings.Contains(normDesc, "1x2"):
			return "FB_CORNERS_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "card") && (strings.Contains(normDesc, "over/under") || strings.Contains(normDesc, "total")):
			return "FB_CARDS_TOTAL", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 2: // Basketball
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline") || strings.Contains(normDesc, "2 way"):
			if strings.Contains(normDesc, "q1") || strings.Contains(normDesc, "1st quarter") {
				return "BB_QUARTER_WINNER", PeriodQuarter1, normSel
			}
			if strings.Contains(normDesc, "q2") || strings.Contains(normDesc, "2nd quarter") {
				return "BB_QUARTER_WINNER", PeriodQuarter2, normSel
			}
			if strings.Contains(normDesc, "q3") || strings.Contains(normDesc, "3rd quarter") {
				return "BB_QUARTER_WINNER", PeriodQuarter3, normSel
			}
			if strings.Contains(normDesc, "q4") || strings.Contains(normDesc, "4th quarter") {
				return "BB_QUARTER_WINNER", PeriodQuarter4, normSel
			}
			if strings.Contains(normDesc, "1st half") || strings.Contains(normDesc, "first half") {
				return "BB_1H_WINNER", PeriodHalfTime, normSel
			}
			return "BB_MONEYLINE", PeriodFullTime, normSel
		case rawID == "18" || strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "total") || strings.Contains(normDesc, "over/under"):
			if strings.Contains(normDesc, "q1") || strings.Contains(normDesc, "1st quarter") {
				return "BB_QUARTER_TOTALS", PeriodQuarter1, normalizeOverUnder(rawSel)
			}
			if strings.Contains(normDesc, "q2") || strings.Contains(normDesc, "2nd quarter") {
				return "BB_QUARTER_TOTALS", PeriodQuarter2, normalizeOverUnder(rawSel)
			}
			if strings.Contains(normDesc, "q3") || strings.Contains(normDesc, "3rd quarter") {
				return "BB_QUARTER_TOTALS", PeriodQuarter3, normalizeOverUnder(rawSel)
			}
			if strings.Contains(normDesc, "q4") || strings.Contains(normDesc, "4th quarter") {
				return "BB_QUARTER_TOTALS", PeriodQuarter4, normalizeOverUnder(rawSel)
			}
			if strings.Contains(normDesc, "1st half") || strings.Contains(normDesc, "first half") {
				return "BB_TOTAL_POINTS", PeriodHalfTime, normalizeOverUnder(rawSel)
			}
			return "BB_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "spread") || strings.Contains(normDesc, "handicap"):
			return "BB_SPREAD", PeriodFullTime, normSel
		}

	case 3: // Tennis
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner") || strings.Contains(normDesc, "2 way"):
			if strings.Contains(normDesc, "set 1") || strings.Contains(normDesc, "1st set") {
				return "TN_SET_WINNER", PeriodSet1, normSel
			}
			if strings.Contains(normDesc, "set 2") || strings.Contains(normDesc, "2nd set") {
				return "TN_SET_WINNER", PeriodSet2, normSel
			}
			if strings.Contains(normDesc, "set 3") || strings.Contains(normDesc, "3rd set") {
				return "TN_SET_WINNER", PeriodSet3, normSel
			}
			return "TN_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total games") || strings.Contains(normDesc, "over/under"):
			return "TN_TOTAL_GAMES", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "game handicap"):
			return "TN_GAME_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "set handicap"):
			return "TN_SET_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "correct score") || strings.Contains(normDesc, "set score"):
			return "TN_CORRECT_SET_SCORE", PeriodFullTime, rawSel
		}

	case 4: // Ice Hockey
		switch {
		case rawID == "1" || strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "3 way result"):
			if strings.Contains(normDesc, "period 1") || strings.Contains(normDesc, "1st period") {
				return "IH_PERIOD_1X2", PeriodPeriod1, normSel
			}
			return "IH_REGULATION_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline") || strings.Contains(normDesc, "2 way"):
			if strings.Contains(normDesc, "period 1") || strings.Contains(normDesc, "1st period") {
				return "IH_PERIOD_1X2", PeriodPeriod1, normSel
			}
			return "IH_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "puck line") || strings.Contains(normDesc, "handicap"):
			return "IH_PUCK_LINE", PeriodFullTime, normSel
		case rawID == "18" || strings.Contains(normDesc, "total goals") || strings.Contains(normDesc, "over/under"):
			if strings.Contains(normDesc, "period 1") || strings.Contains(normDesc, "1st period") {
				return "IH_PERIOD_TOTALS", PeriodPeriod1, normalizeOverUnder(rawSel)
			}
			return "IH_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 5: // Volleyball
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner") || strings.Contains(normDesc, "2 way"):
			if strings.Contains(normDesc, "set 1") || strings.Contains(normDesc, "1st set") {
				return "VB_SET_WINNER", PeriodSet1, normSel
			}
			return "VB_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "over/under"):
			return "VB_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "set handicap"):
			return "VB_SET_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "point handicap"):
			return "VB_POINT_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "correct score"):
			return "VB_CORRECT_SET_SCORE", PeriodFullTime, rawSel
		}

	case 6: // MMA
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "fight winner"):
			return "MM_FIGHT_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "method of victory") || strings.Contains(normDesc, "method"):
			return "MM_METHOD_OF_VICTORY", PeriodFullTime, normalizeMMAMethod(rawSel)
		case strings.Contains(normDesc, "total rounds") || strings.Contains(normDesc, "over/under"):
			return "MM_TOTAL_ROUNDS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "goes distance"):
			return "MM_GOES_DISTANCE", PeriodFullTime, normalizeYesNo(rawSel)
		}

	case 7: // Rugby
		switch {
		case rawID == "1" || strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "3 way result"):
			if strings.Contains(normDesc, "1st half") {
				return "RB_HT_1X2", PeriodHalfTime, normSel
			}
			return "RB_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			return "RB_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "handicap"):
			return "RB_HANDICAP", PeriodFullTime, normSel
		case (strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "over/under")) &&
			!strings.Contains(normDesc, "try") && !strings.Contains(normDesc, "tries"):
			return "RB_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 8: // Cricket
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner"):
			return "CK_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "team runs") || strings.Contains(normDesc, "runs over/under") || strings.Contains(normDesc, "total runs"):
			return "CK_TEAM_RUNS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "most sixes"):
			return "CK_MOST_SIXES", PeriodFullTime, normSel
		case strings.Contains(normDesc, "most fours"):
			return "CK_MOST_FOURS", PeriodFullTime, normSel
		}

	case 9: // American Football
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			if strings.Contains(normDesc, "q1") {
				return "AF_QUARTER_WINNER", PeriodQuarter1, normSel
			}
			return "AF_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "spread") || strings.Contains(normDesc, "handicap"):
			return "AF_SPREAD", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "over/under"):
			return "AF_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 10: // Handball
		switch {
		case rawID == "1" || strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "3 way result"):
			if strings.Contains(normDesc, "1st half") {
				return "HB_HT_1X2", PeriodHalfTime, normSel
			}
			return "HB_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			return "HB_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "handicap"):
			return "HB_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total goals") || strings.Contains(normDesc, "over/under"):
			return "HB_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 11: // Baseball
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			if strings.Contains(normDesc, "first 5") || strings.Contains(normDesc, "f5") {
				return "BS_F5_MONEYLINE", "F5", normSel
			}
			return "BS_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "run line") || strings.Contains(normDesc, "spread"):
			return "BS_RUN_LINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total runs") || strings.Contains(normDesc, "over/under"):
			return "BS_TOTAL_RUNS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 13: // Table Tennis
		switch {
		case rawID == "1" || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner"):
			if strings.Contains(normDesc, "game 1") || strings.Contains(normDesc, "1st game") {
				return "TT_GAME_WINNER", "GAME1", normSel
			}
			return "TT_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "over/under"):
			return "TT_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "point handicap"):
			return "TT_POINT_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "game handicap"):
			return "TT_GAME_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "correct score"):
			return "TT_CORRECT_SCORE", PeriodFullTime, rawSel
		}
	}

	return "", "", ""
}

func mapOdibets(sportID int, rawID, normDesc, rawSel string) (string, string, string) {
	normSel := normalizeSelection(rawSel)

	switch sportID {
	case 1: // Football
		switch {
		case rawID == "1" || rawID == "FB_1X2" || strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "three way") || strings.Contains(normDesc, "match result"):
			return "FB_1X2", PeriodFullTime, normSel
		case (rawID == "18" || rawID == "FB_TOTAL_GOALS" || strings.Contains(normDesc, "over/under") || strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "total goals") || strings.Contains(normDesc, "total")) &&
			!strings.Contains(normDesc, "corner") && !strings.Contains(normDesc, "card") && !strings.Contains(normDesc, "throw") &&
			!strings.Contains(normDesc, "offside") && !strings.Contains(normDesc, "foul") && !strings.Contains(normDesc, "booking"):
			return "FB_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "gg/ng") || strings.Contains(normDesc, "both teams to score") || normDesc == "gg" || normDesc == "btts":
			return "FB_BTTS", PeriodFullTime, normalizeYesNo(rawSel)
		case strings.Contains(normDesc, "double chance"):
			return "FB_DOUBLE_CHANCE", PeriodFullTime, normalizeDoubleChance(rawSel)
		case strings.Contains(normDesc, "draw no bet") || strings.Contains(normDesc, "dnb"):
			return "FB_DNB", PeriodFullTime, normSel
		case strings.Contains(normDesc, "handicap") && !strings.Contains(normDesc, "corner") && !strings.Contains(normDesc, "card") && !strings.Contains(normDesc, "prop"):
			return "FB_ASIAN_HANDICAP", PeriodFullTime, normSel
		case strings.Contains(normDesc, "correct score"):
			return "FB_CORRECT_SCORE", PeriodFullTime, rawSel
		case strings.Contains(normDesc, "half time 1x2") || strings.Contains(normDesc, "1st half 1x2"):
			return "FB_HT_1X2", PeriodHalfTime, normSel
		case strings.Contains(normDesc, "half time total") || strings.Contains(normDesc, "1st half total"):
			return "FB_TOTAL_GOALS", PeriodHalfTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "odd/even"):
			return "FB_ODD_EVEN", PeriodFullTime, normalizeOddEven(rawSel)
		}

	case 2: // Basketball
		switch {
		case strings.Contains(normDesc, "moneyline") || strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "2 way"):
			return "BB_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "total points") || strings.Contains(normDesc, "over/under"):
			return "BB_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		case strings.Contains(normDesc, "spread") || strings.Contains(normDesc, "handicap"):
			return "BB_SPREAD", PeriodFullTime, normSel
		}

	case 3: // Tennis
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner") || strings.Contains(normDesc, "2 way"):
			return "TN_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total games") || strings.Contains(normDesc, "over/under"):
			return "TN_TOTAL_GAMES", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 4: // Ice Hockey
		switch {
		case strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "three way"):
			return "IH_REGULATION_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "moneyline") || strings.Contains(normDesc, "winner"):
			if strings.Contains(normDesc, "period 1") || strings.Contains(normDesc, "1st period") {
				return "IH_PERIOD_1X2", PeriodPeriod1, normSel
			}
			return "IH_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total") || strings.Contains(normDesc, "over/under"):
			return "IH_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 5: // Volleyball
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner"):
			return "VB_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total") || strings.Contains(normDesc, "over/under"):
			return "VB_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 6: // MMA
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "fight winner"):
			return "MM_FIGHT_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "total rounds") || strings.Contains(normDesc, "over/under"):
			return "MM_TOTAL_ROUNDS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 7: // Rugby
		switch {
		case strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "three way"):
			return "RB_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			return "RB_MONEYLINE", PeriodFullTime, normSel
		}

	case 8: // Cricket
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner"):
			return "CK_MATCH_WINNER", PeriodFullTime, normSel
		}

	case 9: // American Football
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			return "AF_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "over/under"):
			return "AF_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 10: // Handball
		switch {
		case strings.Contains(normDesc, "1x2") || strings.Contains(normDesc, "three way"):
			return "HB_1X2", PeriodFullTime, normSel
		case strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "over/under"):
			return "HB_TOTAL_GOALS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 11: // Baseball
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "moneyline"):
			return "BS_MONEYLINE", PeriodFullTime, normSel
		case strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "over/under"):
			return "BS_TOTAL_RUNS", PeriodFullTime, normalizeOverUnder(rawSel)
		}

	case 13: // Table Tennis
		switch {
		case strings.Contains(normDesc, "winner") || strings.Contains(normDesc, "match winner"):
			return "TT_MATCH_WINNER", PeriodFullTime, normSel
		case strings.Contains(normDesc, "totals") || strings.Contains(normDesc, "over/under"):
			return "TT_TOTAL_POINTS", PeriodFullTime, normalizeOverUnder(rawSel)
		}
	}

	return "", "", ""
}

func normalizeSelection(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	switch s {
	case "1", "HOME":
		return "1"
	case "X", "DRAW":
		return "X"
	case "2", "AWAY":
		return "2"
	default:
		return s
	}
}

func normalizeOverUnder(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	if strings.HasPrefix(s, "O") || strings.Contains(s, "OVER") {
		return "OVER"
	}
	if strings.HasPrefix(s, "U") || strings.Contains(s, "UNDER") {
		return "UNDER"
	}
	return s
}

func normalizeYesNo(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	if s == "YES" || s == "GG" || s == "Y" {
		return "YES"
	}
	if s == "NO" || s == "NG" || s == "N" {
		return "NO"
	}
	return s
}

func normalizeDoubleChance(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	s = strings.ReplaceAll(s, " OR ", "")
	s = strings.ReplaceAll(s, "/", "")
	switch s {
	case "1X", "1-X":
		return "1X"
	case "12", "1-2":
		return "12"
	case "X2", "2X", "X-2":
		return "X2"
	default:
		return s
	}
}

func normalizeOddEven(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	if strings.Contains(s, "ODD") {
		return "ODD"
	}
	if strings.Contains(s, "EVEN") {
		return "EVEN"
	}
	return s
}

func normalizeMMAMethod(sel string) string {
	s := strings.ToUpper(strings.TrimSpace(sel))
	switch {
	case strings.Contains(s, "KO") || strings.Contains(s, "TKO"):
		return "KO/TKO"
	case strings.Contains(s, "SUB"):
		return "SUBMISSION"
	case strings.Contains(s, "DEC"):
		return "DECISION"
	default:
		return s
	}
}
