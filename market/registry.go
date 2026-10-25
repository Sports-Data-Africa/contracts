package market

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages canonical settlement rules by key.
type Registry struct {
	mu    sync.RWMutex
	rules map[MarketRuleKey]SettlementRule
}

var defaultRegistry = &Registry{
	rules: make(map[MarketRuleKey]SettlementRule),
}

// Register adds a settlement rule to the default registry.
func Register(rule SettlementRule) {
	defaultRegistry.Register(rule)
}

func (r *Registry) Register(rule SettlementRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := rule.Key()
	// Normalize market code and period to upper-case
	key.MarketCode = strings.ToUpper(strings.TrimSpace(key.MarketCode))
	key.Period = strings.ToUpper(strings.TrimSpace(key.Period))
	if key.RuleVersion <= 0 {
		key.RuleVersion = 1
	}
	r.rules[key] = rule
}

// Get finds a registered settlement rule.
func Get(sportID int, marketCode, period string, ruleVersion int) (SettlementRule, bool) {
	return defaultRegistry.Get(sportID, marketCode, period, ruleVersion)
}

func (r *Registry) Get(sportID int, marketCode, period string, ruleVersion int) (SettlementRule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normCode := strings.ToUpper(strings.TrimSpace(marketCode))
	normPeriod := strings.ToUpper(strings.TrimSpace(period))
	switch normPeriod {
	case "":
		normPeriod = PeriodFullTime
	case "1H", "FIRST_HALF":
		normPeriod = PeriodHalfTime
	case "2H", "SECOND_HALF":
		normPeriod = PeriodSecondHalf
	}
	if ruleVersion <= 0 {
		ruleVersion = 1
	}

	key := MarketRuleKey{
		SportID:     sportID,
		MarketCode:  normCode,
		Period:      normPeriod,
		RuleVersion: ruleVersion,
	}

	rule, ok := r.rules[key]
	return rule, ok
}

// Count returns the total number of registered rules.
func Count() int {
	return defaultRegistry.Count()
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.rules)
}

// ListAll returns all registered rule keys.
func ListAll() []MarketRuleKey {
	return defaultRegistry.ListAll()
}

func (r *Registry) ListAll() []MarketRuleKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]MarketRuleKey, 0, len(r.rules))
	for k := range r.rules {
		keys = append(keys, k)
	}
	return keys
}

// Evaluate evaluates a bet selection using the centrally registered rule.
// Evaluate evaluates a bet selection using the centrally registered rule.
// Returns an error if no rule is registered for the bet's key (Fail-Closed).
func Evaluate(state CanonicalEventState, bet BetSelection) (SettlementResult, error) {
	// 1. Fixture Lifecycle Gating
	switch state.Status {
	case FixtureStatusCancelled:
		// Cancelled fixtures must be VOIDED with full stake refund per house rules, NEVER marked LOST
		return SettlementResult{
			Status:         BetStatusVoid,
			PayoutFactor:   1.0,
			Settled:        true,
			Timing:         TimingEventEnd,
			Reason:         "FIXTURE_CANCELLED: Match officially cancelled; stake fully refunded per sportsbook house rules",
			RequiredDomain: DomainScore,
		}, nil

	case FixtureStatusPostponed:
		// Postponed matches held pending official rescheduling confirmation within policy window
		return SettlementResult{
			Status:         BetStatusSettlementHold,
			PayoutFactor:   0.0,
			Settled:        false,
			Timing:         TimingEventEnd,
			Reason:         "FIXTURE_POSTPONED: Settlement held pending official federation rescheduling determination",
			RequiredDomain: DomainScore,
		}, nil

	case FixtureStatusAbandoned, FixtureStatusInterrupted:
		// Check for unconditionally determined markets:
		// (a) Completed period markets (e.g. HT completed before abandonment at 70')
		normPeriod := NormalizePeriod(bet.Period)
		htScore, hasHT := state.PeriodScores["HT"]
		if normPeriod == PeriodHalfTime && (state.IsPeriodFinished[PeriodHalfTime] || (hasHT && htScore.Home >= 0 && htScore.Away >= 0)) {
			// Period is unconditionally completed! Proceed to regular rule evaluation
			break
		}
		// (b) Over Totals already achieved before abandonment
		if strings.Contains(strings.ToUpper(bet.CanonicalMarketCode), "TOTAL") && strings.EqualFold(strings.TrimSpace(bet.SelectionCode), "OVER") {
			totalScore := state.Score.Home + state.Score.Away
			lineVal := 0.0
			if bet.Line != nil {
				lineVal = *bet.Line
			}
			if float64(totalScore) > lineVal {
				// Unconditionally determined as WON
				return SettlementResult{
					Status:         BetStatusWon,
					PayoutFactor:   1.0,
					Settled:        true,
					Timing:         TimingInstantIrreversible,
					Reason:         fmt.Sprintf("UNCONDITIONALLY_DETERMINED: Total %d exceeded line %.2f before abandonment", totalScore, lineVal),
					RequiredDomain: DomainScore,
				}, nil
			}
		}
		// (c) Undetermined markets on abandoned matches MUST NOT be marked LOST:
		// Held fail-closed pending official abandonment protocol
		return SettlementResult{
			Status:         BetStatusSettlementHold,
			PayoutFactor:   0.0,
			Settled:        false,
			Timing:         TimingEventEnd,
			Reason:         "FIXTURE_ABANDONED: Undetermined market held fail-closed pending official federation ruling",
			RequiredDomain: DomainScore,
		}, nil
	}

	period := bet.Period
	if period == "" {
		return SettlementResult{
			Status:       BetStatusSettlementHold,
			PayoutFactor: 0.0,
			Settled:      false,
			Reason:       fmt.Sprintf("MISSING_PERIOD: Period cannot be inferred during settlement for sport=%d, code=%s", bet.SportID, bet.CanonicalMarketCode),
		}, fmt.Errorf("missing period on bet %s: cannot infer period at settlement", bet.BetID)
	}
	version := bet.RuleVersion
	if version <= 0 {
		version = 1
	}

	rule, exists := Get(bet.SportID, bet.CanonicalMarketCode, period, version)
	if !exists {
		return SettlementResult{
			Status:       BetStatusManualReview,
			PayoutFactor: 0.0,
			Settled:      false,
			Reason:       fmt.Sprintf("NO_RULE_REGISTERED for sport=%d, code=%s, period=%s, v=%d", bet.SportID, bet.CanonicalMarketCode, period, version),
		}, fmt.Errorf("no settlement rule registered for %s in sport %d", bet.CanonicalMarketCode, bet.SportID)
	}

	res := rule.Evaluate(state, bet)
	return res, nil
}
