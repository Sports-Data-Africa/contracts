package canonical

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCanonicalTeamKeyFormattingAndParsing(t *testing.T) {
	testCases := []struct {
		sportID     int
		teamID      int64
		expectedKey string
		sportCode   string
	}{
		{SportIDFootball, 10492, "SPD-FB-10492", "FB"},
		{SportIDBasketball, 2011, "SPD-BB-2011", "BB"},
		{SportIDTennis, 305, "SPD-TN-305", "TN"},
		{SportIDIceHockey, 412, "SPD-IH-412", "IH"},
		{SportIDCricket, 809, "SPD-CK-809", "CK"},
		{SportIDTableTennis, 1301, "SPD-TT-1301", "TT"},
	}

	for _, tc := range testCases {
		key := FormatCanonicalTeamKey(tc.sportID, tc.teamID)
		if key != tc.expectedKey {
			t.Fatalf("expected key %s, got %s", tc.expectedKey, key)
		}

		code, id, err := ParseCanonicalTeamKey(key)
		if err != nil {
			t.Fatalf("unexpected parse error for %s: %v", key, err)
		}
		if code != tc.sportCode || id != tc.teamID {
			t.Fatalf("parse mismatch for %s: got code=%s id=%d; expected code=%s id=%d",
				key, code, id, tc.sportCode, tc.teamID)
		}
	}
}

func TestCanonicalCompetitionKeyFormattingAndParsing(t *testing.T) {
	key := FormatCanonicalCompetitionKey(17)
	if key != "SPD-COMP-17" {
		t.Fatalf("expected SPD-COMP-17, got %s", key)
	}

	id, err := ParseCanonicalCompetitionKey(key)
	if err != nil {
		t.Fatalf("unexpected error parsing %s: %v", key, err)
	}
	if id != 17 {
		t.Fatalf("expected competition ID 17, got %d", id)
	}
}

func TestCanonicalTeamJSONPayloadSerialization(t *testing.T) {
	now := time.Now().UTC()
	team := CanonicalTeam{
		ID:           10492,
		CanonicalKey: "SPD-FB-10492",
		SportID:      1,
		SportCode:    "FB",
		Name:         "Manchester City",
		ShortName:    "MCI",
		CountryCode:  "gb-eng",
		Gender:       "MEN",
		AgeCategory:  "SR",
		Logo: TeamLogoReference{
			URL:             "https://football-logos.cc/logos/premier-league/manchester-city.svg",
			FallbackURL:     "https://a.espncdn.com/i/teamlogos/soccer/500/382.png",
			ProviderLogoURL: "https://static.sportybet.com/sport/icon/teams/10492.png",
			Provenance:      "sportradar",
			LastVerifiedAt:  &now,
		},
		Status:     "ACTIVE",
		IsVerified: true,
		UpdatedAt:  now,
	}

	bytes, err := json.Marshal(team)
	if err != nil {
		t.Fatalf("failed to marshal CanonicalTeam: %v", err)
	}

	var parsed CanonicalTeam
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal CanonicalTeam: %v", err)
	}

	if parsed.ID != 10492 || parsed.Name != "Manchester City" || parsed.ShortName != "MCI" {
		t.Fatalf("mismatch in deserialized team: %+v", parsed)
	}
	if parsed.Logo.URL != team.Logo.URL || parsed.Logo.ProviderLogoURL != team.Logo.ProviderLogoURL {
		t.Fatalf("mismatch in deserialized team logo: %+v", parsed.Logo)
	}
}

func TestTeamReferenceEmbeddingInFixture(t *testing.T) {
	ref := TeamReference{
		ID:          10492,
		CanonicalID: "SDA-FB-10492",
		Name:        "Manchester City",
		ShortName:   "MCI",
		CountryCode: "gb-eng",
		Logo: TeamLogoReference{
			URL:             "https://football-logos.cc/logos/premier-league/manchester-city.svg",
			FallbackURL:     "https://a.espncdn.com/i/teamlogos/soccer/500/382.png",
			ProviderLogoURL: "https://static.sportybet.com/sport/icon/teams/10492.png",
		},
	}

	bytes, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("failed to marshal TeamReference: %v", err)
	}

	var parsed TeamReference
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal TeamReference: %v", err)
	}

	if parsed.CanonicalID != "SDA-FB-10492" || parsed.Name != "Manchester City" {
		t.Fatalf("mismatch in deserialized TeamReference: %+v", parsed)
	}
}
