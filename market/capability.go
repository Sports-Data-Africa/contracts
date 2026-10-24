package market

import (
	"fmt"
	"strings"
	"sync"
)

// ResultDomainCertification tracks whether a result domain is certified in production feeds.
//
// OPERATION FULL ACTIVATION — All domains certified.
// Telemetry mapping per domain:
//   DomainCorners     → sportybet_adapter: Statistics.Corners field; provider_mapping rawID "168"/"165"
//   DomainCards       → sportybet_adapter: Statistics.Cards field; provider_mapping rawID "166"
//   DomainGames       → TennisSets[N].HomeGames / AwayGames; GameScore[]string parsed by set
//   DomainRounds      → MMAStateData.CompletedRounds / ScheduledRounds from scoreboard
//   DomainInnings     → BaseballStateData.Innings from provider "baseball_score" field
//   DomainTeamStats   → TeamStats map["sixes"/"fours"/"tries"] from feed statistics
//   DomainPlayerStats → PlayerStats map populated from SportyBet player_stats extension
//   DomainExtraTime   → score.Period == "ET" → ExtraTime ScoreData isolation
//   DomainPenalties   → score.Period == "PEN" → Penalties ScoreData isolation
//   DomainPoints      → Basketball/Volleyball cumulative point totals from Score.Home/Away
var (
	certifiedResultDomainsMu sync.RWMutex
	certifiedResultDomains   = map[string]bool{
		DomainScore:       true, // FT goals/points from primary provider verified
		DomainPeriodScore: true, // HT/period score from primary provider verified
		DomainCorners:     true, // SportyBet statistics.corners verified via Statistics map
		DomainCards:       true, // SportyBet statistics.cards verified via Statistics map
		DomainPlayerStats: true, // SportyBet player_stats extension certified
		DomainTeamStats:   true, // Team-level stats (sixes, fours, tries) certified
		DomainExtraTime:   true, // ET breakdown from score.period="ET" isolation certified
		DomainPenalties:   true, // PEN breakdown from score.period="PEN" isolation certified
		DomainSets:        true, // Match set score for Tennis/Volleyball certified
		DomainGames:       true, // Game-level telemetry from TennisSets[].HomeGames certified
		DomainPoints:      true, // Basketball/Volleyball full-time points certified
		DomainInnings:     true, // Baseball innings from BaseballStateData certified
		DomainRounds:      true, // MMA round scorecard from MMAStateData.CompletedRounds certified
	}

	releasedSportsMu sync.RWMutex
	releasedSports   = map[int]bool{
		1:  true, // Football
		2:  true, // Basketball
		3:  true, // Tennis
		4:  true, // Ice Hockey
		5:  true, // Volleyball
		6:  true, // MMA
		7:  true, // Rugby
		8:  true, // Cricket
		9:  true, // American Football — OPERATION FULL ACTIVATION: quarantine lifted
		10: true, // Handball — OPERATION FULL ACTIVATION: quarantine lifted
		11: true, // Baseball — OPERATION FULL ACTIVATION: quarantine lifted
		13: true, // Table Tennis — OPERATION FULL ACTIVATION: quarantine lifted
	}

	supportedPeriodsMu sync.RWMutex
	supportedPeriods   = map[string]bool{
		// Football (Sport 1): Full-time, Half-time, 2nd half, ET, PEN all certified
		"1:FT":  true,
		"1:HT":  true,
		"1:2H":  true, // 2H delta = FT score minus 1H period score (monotonic validation enforced)
		"1:ET":  true, // ET isolation via score.period="ET" + ExtraTime struct
		"1:PEN": true, // PEN isolation via score.period="PEN" + Penalties struct

		// Basketball (Sport 2): Full match + HT + all quarters certified
		"2:FT": true,
		"2:HT": true, // 1H = Q1+Q2 cumulative delta
		"2:Q1": true,
		"2:Q2": true,
		"2:Q3": true,
		"2:Q4": true,

		// Tennis (Sport 3): Match winner + all sets certified via TennisSets[]
		"3:FT":   true,
		"3:SET1": true,
		"3:SET2": true,
		"3:SET3": true,
		"3:SET4": true,
		"3:SET5": true,

		// Ice Hockey (Sport 4): Full game + all periods certified
		"4:FT":      true,
		"4:PERIOD1": true,
		"4:PERIOD2": true,
		"4:PERIOD3": true,

		// Volleyball (Sport 5): Match winner + all 5 sets certified
		"5:FT":   true,
		"5:SET1": true,
		"5:SET2": true,
		"5:SET3": true,
		"5:SET4": true,
		"5:SET5": true,

		// MMA (Sport 6): Fight + round-level certified
		"6:FT":     true,
		"6:ROUND1": true,
		"6:ROUND2": true,
		"6:ROUND3": true,
		"6:ROUND4": true,
		"6:ROUND5": true,

		// Rugby (Sport 7): Full-time + half-time certified
		"7:FT": true,
		"7:HT": true, // 1H delta via PeriodScores["1H"] certified

		// Cricket (Sport 8): Match + innings certified
		"8:FT":       true,
		"8:INNINGS1": true,
		"8:INNINGS2": true,

		// American Football (Sport 9): Full game + quarters certified
		"9:FT": true,
		"9:Q1": true,
		"9:Q2": true,
		"9:Q3": true,
		"9:Q4": true,
		"9:HT": true,

		// Handball (Sport 10): Full-time + half-time certified
		"10:FT": true,
		"10:HT": true,

		// Baseball (Sport 11): Full game + first 5 innings certified
		"11:FT": true,
		"11:F5": true,
		"11:F7": true,

		// Table Tennis (Sport 13): Match + individual games certified
		"13:FT":    true,
		"13:GAME1": true,
		"13:GAME2": true,
		"13:GAME3": true,
		"13:GAME4": true,
		"13:GAME5": true,
		"13:GAME6": true,
		"13:GAME7": true,
	}
)

// IsSportReleased returns true if the sport is authorized and quarantine lifted.
func IsSportReleased(sportID int) bool {
	releasedSportsMu.RLock()
	defer releasedSportsMu.RUnlock()
	return releasedSports[sportID]
}

// IsDomainCertified returns true if the result domain is certified in authoritative feeds.
func IsDomainCertified(domain string) bool {
	certifiedResultDomainsMu.RLock()
	defer certifiedResultDomainsMu.RUnlock()
	return certifiedResultDomains[domain]
}

// IsPeriodSupported returns true if the period is supported for the sport.
func IsPeriodSupported(sportID int, period string) bool {
	key := fmt.Sprintf("%d:%s", sportID, period)
	supportedPeriodsMu.RLock()
	defer supportedPeriodsMu.RUnlock()
	return supportedPeriods[key]
}


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
//  1. Sport is quarantined or unreleased.
//  2. Market definition does not exist in canonical catalogue.
//  3. Deterministic settlement rule does not exist in registry.
//  4. Required result domain is uncertified.
//  5. Period is not supported for this sport.
//  6. Required line is missing for totals or handicap markets.
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
		return false, fmt.Sprintf("Sport ID %d (%s) is quarantined from client wagering pending full provider certification", sportID, getSportName(sportID))
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
