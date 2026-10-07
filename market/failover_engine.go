package market

import (
	"fmt"
	"time"
)

// ProviderHealthState tracks the runtime health, latency, and staleness of a sports data provider.
type ProviderHealthState struct {
	Provider            string        `json:"provider"`
	Connected           bool          `json:"connected"`
	LastMessageAt       time.Time     `json:"last_message_at"`
	LastFixtureUpdateAt time.Time     `json:"last_fixture_update_at"`
	LatencyMs           int64         `json:"latency_ms"`
	ParseFailures       int64         `json:"parse_failures"`
	Sequence            int64         `json:"sequence"`
	StaleThreshold      time.Duration `json:"stale_threshold"`
}

// IsHealthy returns true only if provider is connected and not stale.
func (h ProviderHealthState) IsHealthy(now time.Time) bool {
	if !h.Connected {
		return false
	}
	if h.LastMessageAt.IsZero() {
		return false
	}
	staleLimit := h.StaleThreshold
	if staleLimit <= 0 {
		staleLimit = 30 * time.Second
	}
	return now.Sub(h.LastMessageAt) <= staleLimit
}

// DomainResolution represents the outcome of provider primary/failover evaluation for a single domain.
type DomainResolution struct {
	Domain            string              `json:"domain"`
	Resolved          bool                `json:"resolved"`
	SourceProvider    string              `json:"source_provider"` // SPORTYBET, ODIBETS, BOTH_AGREED
	HoldActive        bool                `json:"hold_active"`
	DiscrepancyReason string              `json:"discrepancy_reason,omitempty"`
	EventState        CanonicalEventState `json:"event_state"`
	ResolvedAt        time.Time           `json:"resolved_at"`
}

// FailoverEngine coordinates resolution between Primary (SportyBet) and Failover (Odibets).
type FailoverEngine struct {
	staleThreshold time.Duration
}

func NewFailoverEngine(staleThreshold time.Duration) *FailoverEngine {
	if staleThreshold <= 0 {
		staleThreshold = 30 * time.Second
	}
	return &FailoverEngine{staleThreshold: staleThreshold}
}

// ResolveDomain resolves result for a specific domain according to:
// 1. SportyBet available + valid -> use SportyBet.
// 2. SportyBet unavailable/stale/missing required domain -> use Odibets (Failover).
// 3. Both present and agree -> process normally with highest confidence.
// 4. Both present and materially disagree for required domain -> HOLD ONLY the affected domain/market.
// Unrelated domains settle normally!
func (e *FailoverEngine) ResolveDomain(
	domain string,
	sportyState *CanonicalEventState,
	odiState *CanonicalEventState,
	sportyHealth ProviderHealthState,
	odiHealth ProviderHealthState,
	now time.Time,
) DomainResolution {
	sportyValid := sportyState != nil && sportyHealth.IsHealthy(now) && hasDomainData(domain, *sportyState)
	odiValid := odiState != nil && odiHealth.IsHealthy(now) && hasDomainData(domain, *odiState)

	// Case 1: Both present -> Check for agreement or material discrepancy
	if sportyValid && odiValid {
		discrepant, reason := checkDomainDiscrepancy(domain, *sportyState, *odiState)
		if discrepant {
			// HOLD ONLY the affected domain!
			return DomainResolution{
				Domain:            domain,
				Resolved:          false,
				SourceProvider:    "",
				HoldActive:        true,
				DiscrepancyReason: fmt.Sprintf("DISCREPANCY_BETWEEN_PROVIDERS: %s", reason),
				EventState:        *sportyState,
				ResolvedAt:        now,
			}
		}
		// Agreement -> Settle with highest confidence
		return DomainResolution{
			Domain:         domain,
			Resolved:       true,
			SourceProvider: "BOTH_AGREED",
			HoldActive:     false,
			EventState:     *sportyState,
			ResolvedAt:     now,
		}
	}

	// Case 2: Primary (SportyBet) available and valid
	if sportyValid {
		return DomainResolution{
			Domain:         domain,
			Resolved:       true,
			SourceProvider: ProviderSportyBet,
			HoldActive:     false,
			EventState:     *sportyState,
			ResolvedAt:     now,
		}
	}

	// Case 3: Primary unavailable/stale -> Automatic failover to Odibets
	if odiValid {
		return DomainResolution{
			Domain:         domain,
			Resolved:       true,
			SourceProvider: ProviderOdibets,
			HoldActive:     false,
			EventState:     *odiState,
			ResolvedAt:     now,
		}
	}

	// Case 4: Neither has valid data for this domain -> HOLD
	return DomainResolution{
		Domain:            domain,
		Resolved:          false,
		SourceProvider:    "",
		HoldActive:        true,
		DiscrepancyReason: fmt.Sprintf("NO_HEALTHY_PROVIDER_DATA for %s", domain),
		ResolvedAt:        now,
	}
}

func hasDomainData(domain string, s CanonicalEventState) bool {
	switch domain {
	case DomainScore:
		return s.Score.Home >= 0 && s.Score.Away >= 0
	case DomainPeriodScore:
		return len(s.PeriodScores) > 0
	case DomainCorners:
		_, ok := s.Corners["FT"]
		return ok
	case DomainCards:
		_, ok := s.Cards["FT"]
		return ok
	case DomainSets:
		return s.Score.Home >= 0 && s.Score.Away >= 0
	case DomainGames:
		return len(s.TennisSets) > 0
	case DomainPoints:
		return len(s.PeriodScores) > 0
	case DomainInnings:
		return s.Baseball != nil || len(s.PeriodScores) > 0
	case DomainRounds:
		return s.MMA != nil || s.Score.Home >= 0
	default:
		return s.Score.Home >= 0
	}
}

func checkDomainDiscrepancy(domain string, s CanonicalEventState, o CanonicalEventState) (bool, string) {
	switch domain {
	case DomainScore:
		if s.Score.Home != o.Score.Home || s.Score.Away != o.Score.Away {
			return true, fmt.Sprintf("SportyBet score %d-%d != Odibets score %d-%d", s.Score.Home, s.Score.Away, o.Score.Home, o.Score.Away)
		}
	case DomainPeriodScore:
		for k, v := range s.PeriodScores {
			if oScore, exists := o.PeriodScores[k]; exists {
				if v.Home != oScore.Home || v.Away != oScore.Away {
					return true, fmt.Sprintf("Period %s score %d-%d != %d-%d", k, v.Home, v.Away, oScore.Home, oScore.Away)
				}
			}
		}
	case DomainCorners:
		sc, okS := s.Corners["FT"]
		oc, okO := o.Corners["FT"]
		if okS && okO && (sc.Home != oc.Home || sc.Away != oc.Away) {
			return true, fmt.Sprintf("SportyBet corners %d-%d != Odibets corners %d-%d", sc.Home, sc.Away, oc.Home, oc.Away)
		}
	case DomainCards:
		sCard, okS := s.Cards["FT"]
		oCard, okO := o.Cards["FT"]
		if okS && okO && (sCard.HomeYellow != oCard.HomeYellow || sCard.AwayYellow != oCard.AwayYellow) {
			return true, fmt.Sprintf("Cards discrepancy: SportyBet (%d-%d) != Odibets (%d-%d)", sCard.HomeYellow, sCard.AwayYellow, oCard.HomeYellow, oCard.AwayYellow)
		}
	}
	return false, ""
}
