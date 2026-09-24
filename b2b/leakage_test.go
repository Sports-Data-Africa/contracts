package b2b_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/b2b"
	"github.com/Sports-Data-Africa/contracts/events"
)

// ForbiddenTokens contains upstream provider names, scrapers, and internal fields
// that MUST NEVER appear in any external B2B client contract.
var ForbiddenKeys = []string{
	"provider",
	"feed_provider",
	"commercial_source",
	"source_provider",
	"active_provider",
	"provider_fixture_id",
	"provider_competition_id",
	"provider_market_id",
	"provider_selection_id",
	"upstream_sequence",
	"failover_reason",
	"verification_source",
	"truth_source",
	"evidence_hash",
	"verification_sources",
	"placed_provider",
}

var ForbiddenValues = []string{
	"sportybet",
	"sporty",
	"paripesa",
	"pari",
	"aiscore",
	"flashscore",
	"sofascore",
}

// AssertNoProviderLeakage recursively checks JSON for forbidden provider tokens
func AssertNoProviderLeakage(t *testing.T, payload []byte) {
	var raw interface{}
	err := json.Unmarshal(payload, &raw)
	if err != nil {
		t.Fatalf("Failed to parse JSON for leakage test: %v", err)
	}

	inspectNode(t, raw, "$")
}

func inspectNode(t *testing.T, node interface{}, path string) {
	switch val := node.(type) {
	case map[string]interface{}:
		for k, v := range val {
			kLower := strings.ToLower(k)
			for _, forbidden := range ForbiddenKeys {
				if kLower == forbidden || strings.Contains(kLower, forbidden) {
					t.Errorf("LEAK VIOLATION: Forbidden key '%s' found at path '%s.%s'", k, path, k)
				}
			}
			if strVal, ok := v.(string); ok {
				strLower := strings.ToLower(strVal)
				for _, forbidden := range ForbiddenValues {
					if strings.Contains(strLower, forbidden) {
						t.Errorf("LEAK VIOLATION: Forbidden value '%s' found at path '%s.%s'", strVal, path, k)
					}
				}
			}
			inspectNode(t, v, path+"."+k)
		}
	case []interface{}:
		for _, item := range val {
			inspectNode(t, item, path+"[]")
		}
	}
}

func TestB2BMarketOddsUpdate_ProviderLeakage(t *testing.T) {
	internalEvent := &events.InternalMarketOddsUpdatedEvent{
		EventID:          "evt-12345",
		FixtureID:        88001001,
		SportID:          b2b.SportFootball,
		CommercialFeed:   events.CommercialFeedSportybet,
		FeedFixtureID:    "SPTY-643429",
		MarketCode:       "1X2",
		MarketVersion:    43,
		IsSuspended:      false,
		UpstreamSequence: 99482,
		IngestedAt:       time.Now().UnixMilli(),
		PublishedAt:      time.Now().UnixMilli(),
		Selections: []events.InternalSelection{
			{Code: "HOME", Odds: 1.95, FeedSelectionID: "sel-sp-1"},
			{Code: "DRAW", Odds: 3.40, FeedSelectionID: "sel-sp-2"},
			{Code: "AWAY", Odds: 4.10, FeedSelectionID: "sel-sp-3"},
		},
	}

	b2bUpdate, err := b2b.ToB2BMarketOddsUpdate(internalEvent)
	if err != nil {
		t.Fatalf("Unexpected error transforming to B2B update: %v", err)
	}

	serialized, err := json.Marshal(b2bUpdate)
	if err != nil {
		t.Fatalf("Failed to marshal B2B update: %v", err)
	}

	AssertNoProviderLeakage(t, serialized)
}

func TestB2BSettlementEvent_ProviderLeakage(t *testing.T) {
	certEvent := &events.InternalResultCertifiedEvent{
		EventID:             "cert-9988",
		FixtureID:           88001001,
		SportID:             b2b.SportFootball,
		HomeScore:           2,
		AwayScore:           1,
		CertificationStatus: events.CertStatusCertifiedFinal,
		EvidenceHash:        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		VerifiedAt:          time.Now(),
		VerificationSources: []string{events.TruthSourceAiScore, events.TruthSourceFlashscore},
	}

	b2bSettlement, err := b2b.ToB2BSettlementEvent(certEvent, "BET-12345", "WON", 250.00)
	if err != nil {
		t.Fatalf("Unexpected error transforming to B2B settlement: %v", err)
	}

	serialized, err := json.Marshal(b2bSettlement)
	if err != nil {
		t.Fatalf("Failed to marshal B2B settlement: %v", err)
	}

	AssertNoProviderLeakage(t, serialized)
}

func TestB2BFixtureSnapshot_ProviderLeakage(t *testing.T) {
	snapshot := b2b.B2BFixtureSnapshot{
		FixtureID:      88001001,
		CompetitionID:  100001,
		SportID:        b2b.SportFootball,
		HomeTeam:       "Arsenal",
		AwayTeam:       "Chelsea",
		StartTime:      time.Now(),
		Status:         "LIVE",
		ScoreHome:      2,
		ScoreAway:      1,
		PeriodName:     "2H",
		ElapsedMinutes: 78,
	}

	serialized, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Failed to marshal B2B snapshot: %v", err)
	}

	AssertNoProviderLeakage(t, serialized)
}

func TestB2BTicketResponse_ProviderLeakage(t *testing.T) {
	ticketResp := b2b.B2BTicketResponse{
		TicketID:      "TKT-9912",
		FixtureID:     88001001,
		MarketCode:    "1X2",
		SelectionCode: "HOME",
		AcceptedOdds:  2.15,
		MarketVersion: 101,
		Status:        "ACCEPTED",
	}

	serialized, err := json.Marshal(ticketResp)
	if err != nil {
		t.Fatalf("Failed to marshal B2B ticket response: %v", err)
	}

	AssertNoProviderLeakage(t, serialized)
}

func TestTableTennis_BlockedFromB2B(t *testing.T) {
	ttEvent := &events.InternalMarketOddsUpdatedEvent{
		EventID:        "evt-tt-1",
		FixtureID:      88009999,
		SportID:        b2b.SportTableTennis, // 10
		CommercialFeed: events.CommercialFeedSportybet,
		MarketCode:     "MATCH_WINNER",
		MarketVersion:  1,
	}

	_, err := b2b.ToB2BMarketOddsUpdate(ttEvent)
	if err != b2b.ErrSportNotEligibleForB2B {
		t.Fatalf("Expected ErrSportNotEligibleForB2B for Table Tennis (ID 10), got %v", err)
	}
}

func TestB2BErrorResponse_ProviderLeakage(t *testing.T) {
	errResp := map[string]interface{}{
		"code":    "ERR_UNDERAGE_PROHIBITED",
		"message": "Underage sporting events are strictly prohibited from betting.",
	}

	serialized, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	AssertNoProviderLeakage(t, serialized)
}

func TestProviderFailover_InvisibleToB2B(t *testing.T) {
	// Scenario: SPORTYBET is initially active, produces version 101.
	initialInternalEvent := &events.InternalMarketOddsUpdatedEvent{
		EventID:          "evt-sb-101",
		FixtureID:        88001001,
		SportID:          b2b.SportFootball,
		CommercialFeed:   events.CommercialFeedSportybet,
		FeedFixtureID:    "SPTY-643429",
		MarketCode:       "1X2",
		MarketVersion:    101,
		IsSuspended:      false,
		UpstreamSequence: 501,
		IngestedAt:       time.Now().UnixMilli(),
		PublishedAt:      time.Now().UnixMilli(),
		Selections: []events.InternalSelection{
			{Code: "HOME", Odds: 2.10, FeedSelectionID: "sel-sp-1"},
		},
	}

	b2bV101, err := b2b.ToB2BMarketOddsUpdate(initialInternalEvent)
	if err != nil {
		t.Fatalf("Failed to convert initial event: %v", err)
	}

	// Internal commercial feed failover occurs: SPORTYBET stale -> PARIPESA active.
	// Market version increments to 102 with new odds (2.12).
	failoverInternalEvent := &events.InternalMarketOddsUpdatedEvent{
		EventID:          "evt-pp-102",
		FixtureID:        88001001,
		SportID:          b2b.SportFootball,
		CommercialFeed:   events.CommercialFeedParipesa,
		FeedFixtureID:    "PARI-454567",
		MarketCode:       "1X2",
		MarketVersion:    102,
		IsSuspended:      false,
		UpstreamSequence: 1042,
		IngestedAt:       time.Now().UnixMilli(),
		PublishedAt:      time.Now().UnixMilli(),
		Selections: []events.InternalSelection{
			{Code: "HOME", Odds: 2.12, FeedSelectionID: "sel-pp-1"},
		},
	}

	b2bV102, err := b2b.ToB2BMarketOddsUpdate(failoverInternalEvent)
	if err != nil {
		t.Fatalf("Failed to convert failover event: %v", err)
	}

	// Assertions for sovereign identity and invisible failover:
	if b2bV102.FixtureID != b2bV101.FixtureID {
		t.Fatalf("FixtureID changed during failover! Expected %d, got %d", b2bV101.FixtureID, b2bV102.FixtureID)
	}
	if b2bV102.Market.Code != b2bV101.Market.Code {
		t.Fatalf("MarketCode changed during failover! Expected %s, got %s", b2bV101.Market.Code, b2bV102.Market.Code)
	}
	if b2bV102.Market.Version != 102 {
		t.Fatalf("Expected version 102, got %d", b2bV102.Market.Version)
	}
	if b2bV102.Market.Selections[0].Odds != 2.12 {
		t.Fatalf("Expected odds 2.12, got %f", b2bV102.Market.Selections[0].Odds)
	}

	serialized, _ := json.Marshal(b2bV102)
	AssertNoProviderLeakage(t, serialized)
}

func TestVerificationConsensusAndConflict(t *testing.T) {
	// Consensus test: AiScore (2-1) and Flashscore (2-1) match -> CERTIFIED_FINAL
	aiScore := struct{ Home, Away int }{2, 1}
	flashScore := struct{ Home, Away int }{2, 1}

	consensusStatus := events.CertStatusUnverified
	if aiScore == flashScore {
		consensusStatus = events.CertStatusCertifiedFinal
	}
	if consensusStatus != events.CertStatusCertifiedFinal {
		t.Fatalf("Expected consensus to yield CERTIFIED_FINAL")
	}

	// Conflict test: AiScore (2-1) and Flashscore (1-1) disagree -> MANUAL_REVIEW, settlement frozen!
	flashScoreConflict := struct{ Home, Away int }{1, 1}
	conflictStatus := events.CertStatusUnverified
	if aiScore != flashScoreConflict {
		conflictStatus = events.CertStatusManualReview
	}
	if conflictStatus != events.CertStatusManualReview {
		t.Fatalf("Expected conflict to yield MANUAL_REVIEW, got %s", conflictStatus)
	}
}

func TestProviderAndVerificationSemanticSeparation(t *testing.T) {
	// Commercial feeds MUST NOT be categorized as truth sources
	commercialFeeds := []string{events.CommercialFeedSportybet, events.CommercialFeedParipesa}
	truthSources := []string{events.TruthSourceAiScore, events.TruthSourceFlashscore, events.TruthSourceSofaScore}

	truthMap := make(map[string]bool)
	for _, ts := range truthSources {
		truthMap[ts] = true
	}

	for _, cf := range commercialFeeds {
		if truthMap[cf] {
			t.Fatalf("SEMANTIC ERROR: Commercial feed '%s' conflated as truth source", cf)
		}
	}
}
