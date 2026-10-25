package canonical

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Canonical Internal Provider Constants
const (
	ProviderSportsXBet = "SPORTSXBET"
	ProviderOddixBets  = "ODDIXBETS"

	// Display Names for UI, Logs, and Forensic Audit Exports
	DisplayNameSportsXBet = "SportsXBet"
	DisplayNameOddixBets  = "OddixBets"

	// Stable Internal Provider IDs
	ProviderIDSportsXBet = 1
	ProviderIDOddixBets  = 2

	// Legacy aliases (Deprecated: Retained strictly for backward-compatible normalization)
	LegacyProviderSportyBet = "SPORTYBET"
	LegacyProviderOdibets   = "ODIBETS"
)

// ProviderInfo holds canonical metadata and operational settings for an integrated provider.
type ProviderInfo struct {
	ID                 int      `json:"id"`
	Code               string   `json:"code"`                // "SPORTSXBET", "ODDIXBETS"
	DisplayName        string   `json:"display_name"`        // "SportsXBet", "OddixBets"
	UpstreamIdentifier string   `json:"upstream_identifier"` // "sportybet", "odibets" (upstream protocol/wire ID)
	EnvPrefix          string   `json:"env_prefix"`          // "SPORTSXBET_", "ODDIXBETS_"
	LegacyAliases      []string `json:"legacy_aliases"`
	IsEnabled          bool     `json:"is_enabled"`
	SupportedSports    []int    `json:"supported_sports"`
	Description        string   `json:"description"`
}

// ProviderProvenance records end-to-end audit traceability from raw wire ingestion to settlement.
type ProviderProvenance struct {
	InternalProviderID    int       `json:"internal_provider_id"`
	CanonicalProviderName string    `json:"canonical_provider_name"` // "SPORTSXBET" or "ODDIXBETS"
	DisplayName           string    `json:"display_name"`            // "SportsXBet" or "OddixBets"
	UpstreamProviderID    string    `json:"upstream_provider_id"`    // e.g. "sportybet" or "odibets"
	RawEventID            string    `json:"raw_event_id"`            // e.g. "sr:match:12345"
	RawMarketID           string    `json:"raw_market_id"`           // e.g. "18"
	RawOutcomeID          string    `json:"raw_outcome_id"`          // e.g. "total_over"
	RawSpecifiers         string    `json:"raw_specifiers,omitempty"`
	IngestedAt            time.Time `json:"ingested_at"`
	NormalizedAt          time.Time `json:"normalized_at"`
	MappingVersion        int       `json:"mapping_version"`
	RuleVersion           int       `json:"rule_version,omitempty"`
	CorrelationID         string    `json:"correlation_id,omitempty"`
}

// ProviderRegistry provides centralized, thread-safe provider lookup and normalization.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]ProviderInfo // Keyed by canonical uppercase code
	aliasMap  map[string]string       // Keyed by uppercase alias -> canonical code
}

var (
	defaultRegistry     *ProviderRegistry
	defaultRegistryOnce sync.Once
)

// DefaultProviderRegistry returns the singleton canonical provider registry.
func DefaultProviderRegistry() *ProviderRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = &ProviderRegistry{
			providers: make(map[string]ProviderInfo),
			aliasMap:  make(map[string]string),
		}

		// Register SportsXBet (formerly SportyBet)
		defaultRegistry.registerInternal(ProviderInfo{
			ID:                 ProviderIDSportsXBet,
			Code:               ProviderSportsXBet,
			DisplayName:        DisplayNameSportsXBet,
			UpstreamIdentifier: "sportybet",
			EnvPrefix:          "SPORTSXBET_",
			LegacyAliases:      []string{LegacyProviderSportyBet}, // Exactly ONE legacy alias: "SPORTYBET"
			IsEnabled:          true,
			SupportedSports:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13},
			Description:        "Primary commercial odds and prematch/live feed provider",
		})

		// Register OddixBets (formerly Odibets)
		defaultRegistry.registerInternal(ProviderInfo{
			ID:                 ProviderIDOddixBets,
			Code:               ProviderOddixBets,
			DisplayName:        DisplayNameOddixBets,
			UpstreamIdentifier: "odibets",
			EnvPrefix:          "ODDIXBETS_",
			LegacyAliases:      []string{LegacyProviderOdibets}, // Exactly ONE legacy alias: "ODIBETS"
			IsEnabled:          true,
			SupportedSports:    []int{1, 2, 3, 7},
			Description:        "Secondary commercial feed and failover odds provider",
		})
	})
	return defaultRegistry
}

func (r *ProviderRegistry) registerInternal(info ProviderInfo) {
	code := strings.ToUpper(strings.TrimSpace(info.Code))
	r.providers[code] = info
	r.aliasMap[code] = code

	for _, alias := range info.LegacyAliases {
		normAlias := strings.ToUpper(strings.TrimSpace(alias))
		r.aliasMap[normAlias] = code
	}
}

// NormalizeProvider strictly maps a provider string or its single legacy alias to canonical code.
// Strictly deterministic: NO heuristic guesswork or fuzzy substring matches tolerated.
// - "SPORTSXBET" -> "SPORTSXBET"
// - "SPORTYBET"  -> "SPORTSXBET" (Exact single legacy alias)
// - "ODDIXBETS"  -> "ODDIXBETS"
// - "ODIBETS"    -> "ODDIXBETS"  (Exact single legacy alias)
func NormalizeProvider(raw string) string {
	cleaned := strings.ToUpper(strings.TrimSpace(raw))
	switch cleaned {
	case ProviderSportsXBet, LegacyProviderSportyBet:
		return ProviderSportsXBet
	case ProviderOddixBets, LegacyProviderOdibets:
		return ProviderOddixBets
	default:
		return cleaned
	}
}

// GetProviderInfo returns the metadata for a canonical provider code or alias.
func GetProviderInfo(provider string) (ProviderInfo, bool) {
	reg := DefaultProviderRegistry()
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	canonicalCode := NormalizeProvider(provider)
	info, exists := reg.providers[canonicalCode]
	return info, exists
}

// GetProviderDisplayName returns the human-readable display name (e.g. "SportsXBet").
func GetProviderDisplayName(provider string) string {
	if info, exists := GetProviderInfo(provider); exists {
		return info.DisplayName
	}
	norm := NormalizeProvider(provider)
	if norm == ProviderSportsXBet {
		return DisplayNameSportsXBet
	}
	if norm == ProviderOddixBets {
		return DisplayNameOddixBets
	}
	return provider
}

// GetInternalProviderID returns the stable integer provider ID (1 for SportsXBet, 2 for OddixBets).
func GetInternalProviderID(provider string) int {
	if info, exists := GetProviderInfo(provider); exists {
		return info.ID
	}
	norm := NormalizeProvider(provider)
	if norm == ProviderSportsXBet {
		return ProviderIDSportsXBet
	}
	if norm == ProviderOddixBets {
		return ProviderIDOddixBets
	}
	return 0
}

// CreateProvenance constructs a verified ProviderProvenance record for audit trails.
func CreateProvenance(
	provider, upstreamID, rawEventID, rawMarketID, rawOutcomeID, rawSpecifiers string,
	ingestedAt time.Time, mappingVersion int, correlationID string,
) ProviderProvenance {
	norm := NormalizeProvider(provider)
	disp := GetProviderDisplayName(norm)
	pid := GetInternalProviderID(norm)

	if upstreamID == "" {
		switch norm {
		case ProviderSportsXBet:
			upstreamID = "sportybet"
		case ProviderOddixBets:
			upstreamID = "odibets"
		default:
			upstreamID = strings.ToLower(provider)
		}
	}

	now := time.Now().UTC()
	if ingestedAt.IsZero() {
		ingestedAt = now
	}

	return ProviderProvenance{
		InternalProviderID:    pid,
		CanonicalProviderName: norm,
		DisplayName:           disp,
		UpstreamProviderID:    upstreamID,
		RawEventID:            rawEventID,
		RawMarketID:           rawMarketID,
		RawOutcomeID:          rawOutcomeID,
		RawSpecifiers:         rawSpecifiers,
		IngestedAt:            ingestedAt,
		NormalizedAt:          now,
		MappingVersion:        mappingVersion,
		CorrelationID:         correlationID,
	}
}

// ValidateProviderActive verifies if a provider is registered and authorized.
func ValidateProviderActive(provider string) error {
	norm := NormalizeProvider(provider)
	info, exists := GetProviderInfo(norm)
	if !exists {
		return fmt.Errorf("PROVIDER_UNKNOWN: Provider '%s' is not registered in canonical registry", provider)
	}
	if !info.IsEnabled {
		return fmt.Errorf("PROVIDER_DISABLED: Provider '%s' (%s) is disabled", info.DisplayName, info.Code)
	}
	return nil
}
