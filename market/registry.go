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
	if normPeriod == "" {
		normPeriod = PeriodFullTime
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
// Returns an error if no rule is registered for the bet's key (Fail-Closed).
func Evaluate(state CanonicalEventState, bet BetSelection) (SettlementResult, error) {
	period := bet.Period
	if period == "" {
		period = PeriodFullTime
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
