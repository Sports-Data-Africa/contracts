package market

import (
	"fmt"
	"strings"
	"sync"
)

// ResultDomainCertification tracks whether a result domain is certified in production feeds.
var (
	certifiedResultDomainsMu sync.RWMutex
	certifiedResultDomains   = map[string]bool{
		DomainScore:       true, // FT goals/points from primary provider verified
		DomainPeriodScore: true, // HT period score from primary provider verified (Football HT certified)
		DomainCorners:     false, // Feed statistics wire not yet certified for financial settlement
		DomainCards:       false, // Feed statistics wire not yet certified for financial settlement
		DomainPlayerStats: false, // Player prop feeds not certified
		DomainTeamStats:   false, // Team prop feeds not certified
		DomainExtraTime:   false, // ET breakdown feeds not certified
		DomainPenalties:   false, // Penalty shootout breakdown feeds not certified
		DomainSets:        true,  // Match set score for Tennis/Volleyball match winner certified
		DomainGames:       false, // Game-level telemetry not certified
		DomainPoints:      true,  // Basketball full-time points certified
		DomainInnings:     false, // Cricket ball-by-ball not certified
		DomainRounds:      false, // MMA round scorecard not certified
	}

	releasedSportsMu sync.RWMutex
	releasedSports   = map[int]bool{
		1: true,  // Football
		2: true,  // Basketball
		3: true,  // Tennis
		4: true,  // Ice Hockey
		5: true,  // Volleyball
		6: true,  // MMA
		7: true,  // Rugby
		8: true,  // Cricket
		9: false, // American Football (Quarantined)
		10: false, // Handball (Quarantined)
		11: false, // Baseball (Quarantined)
		13: false, // Table Tennis (Quarantined)
	}

	supportedPeriodsMu sync.RWMutex
	supportedPeriods   = map[string]bool{
		// Football (Sport 1): Full-time and Half-time certified
		"1:FT": true,
		"1:HT": true,
		"1:2H": false, // 2nd half breakdown wire suspended pending dedicated 2H feed
		"1:ET": false,
		"1:PEN": false,

		// Basketball (Sport 2): Full match (incl OT) certified; quarters suspended
		"2:FT": true,
		"2:HT": false,
		"2:Q1": false,
		"2:Q2": false,
		"2:Q3": false,
		"2:Q4": false,

		// Tennis (Sport 3): Match winner certified; individual sets suspended
		"3:FT":   true,
		"3:SET1": false,
		"3:SET2": false,
		"3:SET3": false,

		// Ice Hockey (Sport 4): Full game (incl OT/SO) certified; periods suspended
		"4:FT":      true,
		"4:PERIOD1": false,
		"4:PERIOD2": false,
		"4:PERIOD3": false,

		// Volleyball (Sport 5): Match winner certified
		"5:FT":   true,
		"5:SET1": false,

		// MMA (Sport 6): Fight winner certified
		"6:FT": true,

		// Rugby (Sport 7): Full-time match certified
		"7:FT": true,
		"7:HT": false,

		// Cricket (Sport 8): Match winner certified
		"8:FT": true,
	}
)

// CapabilityItem represents a single market capability in the platform registry.
type CapabilityItem struct {
	SportID              int    `json:"sport_id"`
	SportName            string `json:"sport_name"`
	CanonicalMarketCode  string `json:"canonical_market_code"`
	MarketName           string `json:"market_name"`
	Period               string `json:"period"`
	FormulaType          string `json:"formula_type"`
	Metric               string `json:"metric"`
	Target               string `json:"target"`
	RequiredResultDomain string `json:"required_result_domain"`
	DomainCertified      bool   `json:"domain_certified"`
	PeriodSupported      bool   `json:"period_supported"`
	SportReleased        bool   `json:"sport_released"`
	RuleVersion          int    `json:"rule_version"`
	HasSettlementRule    bool   `json:"has_settlement_rule"`
	Status               string `json:"status"` // ACTIVE, SUSPENDED, QUARANTINED, DISABLED_UNSUPPORTED
	Reason               string `json:"reason,omitempty"`
}

// CapabilityMatrixReport is the operator-facing report generated from the authoritative registry.
type CapabilityMatrixReport struct {
	TotalCapabilities int              `json:"total_capabilities"`
	ActiveCount       int              `json:"active_count"`
	SuspendedCount    int              `json:"suspended_count"`
	QuarantinedCount  int              `json:"quarantined_count"`
	DisabledCount     int              `json:"disabled_count"`
	Capabilities      []CapabilityItem `json:"capabilities"`
}

// MarketPlacementRequest represents a market to be validated for client betting admission.
type MarketPlacementRequest struct {
	SportID             int      `json:"sport_id"`
	CanonicalMarketCode string   `json:"canonical_market_code"`
	Period              string   `json:"period"`
	RawMarketName       string   `json:"raw_market_name,omitempty"`
	Line                *float64 `json:"line,omitempty"`
	RuleVersion         int      `json:"rule_version,omitempty"`
}

// CanPlaceMarketCanonical evaluates whether a market can be accepted for placement using the canonical capability registry.
// Fails closed if:
//   1. Sport is quarantined or unreleased.
//   2. Market definition does not exist in canonical catalogue.
//   3. Deterministic settlement rule does not exist in registry.
//   4. Required result domain is uncertified.
//   5. Period is not supported for this sport.
//   6. Required line is missing for totals or handicap markets.
func CanPlaceMarketCanonical(req MarketPlacementRequest) (bool, string) {
	sportID := req.SportID
	marketCode := strings.ToUpper(strings.TrimSpace(req.CanonicalMarketCode))
	period := strings.ToUpper(strings.TrimSpace(req.Period))
	if period == "" {
		period = PeriodFullTime
	}
	version := req.RuleVersion
	if version <= 0 {
		version = 1
	}

	// 1. Sport Release Check
	releasedSportsMu.RLock()
	released, ok := releasedSports[sportID]
	releasedSportsMu.RUnlock()
	if !ok || !released {
		return false, fmt.Sprintf("Sport ID %d is quarantined from client wagering pending full provider certification", sportID)
	}

	// 2. Active Canonical Catalogue Definition Check
	def, exists := GetMarketDefinition(sportID, marketCode, period)
	if !exists {
		return false, fmt.Sprintf("Market '%s' (Sport %d, Period %s) does not exist in canonical catalogue", marketCode, sportID, period)
	}
	if def.Status != MarketStatusActive {
		return false, fmt.Sprintf("Market '%s' (Sport %d, Period %s) is currently %s", marketCode, sportID, period, def.Status)
	}

	// 3. Deterministic Settlement Rule Registration Check
	rule, hasRule := Get(sportID, marketCode, period, version)
	if !hasRule {
		return false, fmt.Sprintf("No deterministic settlement rule registered for market '%s' (Sport %d, Period %s, v%d)", marketCode, sportID, period, version)
	}

	// 4. Authoritative Result Domain Certification Check
	certifiedResultDomainsMu.RLock()
	domainCertified, domainKnown := certifiedResultDomains[rule.RequiredDomain()]
	certifiedResultDomainsMu.RUnlock()
	if !domainKnown || !domainCertified {
		return false, fmt.Sprintf("Market '%s' is suspended: required result domain '%s' is not yet certified in authoritative feeds", marketCode, rule.RequiredDomain())
	}

	// 5. Independent Period Support Check
	periodKey := fmt.Sprintf("%d:%s", sportID, period)
	supportedPeriodsMu.RLock()
	periodSupported, periodKnown := supportedPeriods[periodKey]
	supportedPeriodsMu.RUnlock()
	if !periodKnown || !periodSupported {
		return false, fmt.Sprintf("Period '%s' for Sport %d is suspended pending verified period feed delivery", period, sportID)
	}

	// 6. Deterministic Line Requirement
	if def.HasLine && req.Line == nil {
		return false, fmt.Sprintf("Market '%s' requires a deterministic line parameter", marketCode)
	}

	return true, ""
}

// GenerateCapabilityMatrix produces the authoritative capability matrix across all catalogued markets.
func GenerateCapabilityMatrix() CapabilityMatrixReport {
	catalogueMu.RLock()
	defer catalogueMu.RUnlock()

	var items []CapabilityItem
	var active, suspended, quarantined, disabled int

	for _, def := range catalogue {
		sportID := def.SportID
		mktCode := def.MarketCode
		period := def.Period
		version := def.RuleVersion
		if version <= 0 {
			version = 1
		}

		sportName := getSportName(sportID)

		releasedSportsMu.RLock()
		isReleased := releasedSports[sportID]
		releasedSportsMu.RUnlock()

		rule, hasRule := Get(sportID, mktCode, period, version)

		domain := def.RequiredResultDomain
		if hasRule {
			domain = rule.RequiredDomain()
		}

		certifiedResultDomainsMu.RLock()
		domainCertified := certifiedResultDomains[domain]
		certifiedResultDomainsMu.RUnlock()

		periodKey := fmt.Sprintf("%d:%s", sportID, period)
		supportedPeriodsMu.RLock()
		periodSupported := supportedPeriods[periodKey]
		supportedPeriodsMu.RUnlock()

		formula := def.FormulaType
		metric := def.Metric
		target := def.Target
		if formula == "" {
			formula, metric, target = inferFormulaMetricTarget(mktCode)
		}

		status := MarketStatusActive
		var reason string

		if !isReleased {
			status = MarketStatusQuarantined
			reason = fmt.Sprintf("Sport %s (ID %d) is quarantined pending official feed certification", sportName, sportID)
		} else if !hasRule {
			status = MarketStatusDisabledUnsupported
			reason = "No settlement rule registered"
		} else if !domainCertified {
			status = MarketStatusSuspended
			reason = fmt.Sprintf("Result domain %s not certified", domain)
		} else if !periodSupported {
			status = MarketStatusSuspended
			reason = fmt.Sprintf("Period %s not certified for sport %d", period, sportID)
		} else if def.Status != MarketStatusActive {
			status = def.Status
			reason = "Catalogue definition marked inactive"
		}

		switch status {
		case MarketStatusActive:
			active++
		case MarketStatusSuspended:
			suspended++
		case MarketStatusQuarantined:
			quarantined++
		default:
			disabled++
		}

		items = append(items, CapabilityItem{
			SportID:              sportID,
			SportName:            sportName,
			CanonicalMarketCode:  mktCode,
			MarketName:           def.MarketName,
			Period:               period,
			FormulaType:          formula,
			Metric:               metric,
			Target:               target,
			RequiredResultDomain: domain,
			DomainCertified:      domainCertified,
			PeriodSupported:      periodSupported,
			SportReleased:        isReleased,
			RuleVersion:          version,
			HasSettlementRule:    hasRule,
			Status:               status,
			Reason:               reason,
		})
	}

	return CapabilityMatrixReport{
		TotalCapabilities: len(items),
		ActiveCount:       active,
		SuspendedCount:    suspended,
		QuarantinedCount:  quarantined,
		DisabledCount:     disabled,
		Capabilities:      items,
	}
}

func getSportName(sportID int) string {
	switch sportID {
	case 1:
		return "Football"
	case 2:
		return "Basketball"
	case 3:
		return "Tennis"
	case 4:
		return "Ice Hockey"
	case 5:
		return "Volleyball"
	case 6:
		return "MMA"
	case 7:
		return "Rugby"
	case 8:
		return "Cricket"
	case 9:
		return "American Football"
	case 10:
		return "Handball"
	case 11:
		return "Baseball"
	case 13:
		return "Table Tennis"
	default:
		return fmt.Sprintf("Sport_%d", sportID)
	}
}

func inferFormulaMetricTarget(code string) (string, string, string) {
	c := strings.ToUpper(code)
	switch {
	case strings.Contains(c, "1X2") || strings.Contains(c, "MATCH_RESULT"):
		return Formula1X2, MetricGoals, TargetMatch
	case strings.Contains(c, "MONEYLINE") || strings.Contains(c, "WINNER"):
		return FormulaMoneyline, MetricGoals, TargetMatch
	case strings.Contains(c, "TOTAL") || strings.Contains(c, "OVER_UNDER"):
		if strings.Contains(c, "HOME") {
			return FormulaTotals, MetricGoals, TargetHome
		}
		if strings.Contains(c, "AWAY") {
			return FormulaTotals, MetricGoals, TargetAway
		}
		return FormulaTotals, MetricGoals, TargetMatch
	case strings.Contains(c, "HANDICAP") || strings.Contains(c, "SPREAD"):
		return FormulaHandicap, MetricGoals, TargetMatch
	case strings.Contains(c, "BTTS") || strings.Contains(c, "GG_NG"):
		return FormulaBTTS, MetricGoals, TargetMatch
	case strings.Contains(c, "DOUBLE_CHANCE"):
		return FormulaDoubleChance, MetricGoals, TargetMatch
	case strings.Contains(c, "DNB") || strings.Contains(c, "DRAW_NO_BET"):
		return FormulaDNB, MetricGoals, TargetMatch
	case strings.Contains(c, "ODD_EVEN"):
		return FormulaOddEven, MetricGoals, TargetMatch
	case strings.Contains(c, "CORRECT_SCORE") || strings.Contains(c, "CLEAN_SHEET") || strings.Contains(c, "WIN_TO_NIL"):
		return FormulaExact, MetricGoals, TargetMatch
	default:
		return Formula1X2, MetricGoals, TargetMatch
	}
}
