package canonical

import (
	"testing"
	"time"
)

func TestCanonicalFixtureIDFormatting(t *testing.T) {
	tests := []struct {
		sportID  int
		seq      int64
		expected string
	}{
		{SportIDFootball, 456778, "SPD-FB-456778"},
		{SportIDBasketball, 293841, "SPD-BB-293841"},
		{SportIDTennis, 582910, "SPD-TN-582910"},
		{SportIDCricket, 682910, "SPD-CK-682910"},
		{SportIDIceHockey, 12, "SPD-IH-000012"},
		{SportIDTableTennis, 999, "SPD-TT-000999"},
	}

	for _, tt := range tests {
		got, err := FormatCanonicalFixtureID(tt.sportID, tt.seq)
		if err != nil {
			t.Fatalf("unexpected error formatting sport %d, seq %d: %v", tt.sportID, tt.seq, err)
		}
		if got != tt.expected {
			t.Errorf("FormatCanonicalFixtureID(%d, %d) = %s, expected %s", tt.sportID, tt.seq, got, tt.expected)
		}
	}
}

func TestCanonicalFixtureIDParsing(t *testing.T) {
	tests := []struct {
		input       string
		expectedSid int
		expectedCod string
		expectedSeq int64
		shouldErr   bool
	}{
		{"SPD-FB-456778", SportIDFootball, "FB", 456778, false},
		{"SPD-BB-293841", SportIDBasketball, "BB", 293841, false},
		{"SPD-TN-000123", SportIDTennis, "TN", 123, false},
		{"SPD-CK-999999", SportIDCricket, "CK", 999999, false},
		{"INVALID-ID", 0, "", 0, true},
		{"SPD-XX-123456", 0, "", 0, true},
		{"SPD-FB-12345", 0, "", 0, true}, // 5 digits
		{"SPD-FB-1234567", 0, "", 0, true}, // 7 digits
	}

	for _, tt := range tests {
		sid, cod, seq, err := ParseCanonicalFixtureID(tt.input)
		if tt.shouldErr {
			if err == nil {
				t.Errorf("expected error for input %s, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
			if sid != tt.expectedSid || cod != tt.expectedCod || seq != tt.expectedSeq {
				t.Errorf("ParseCanonicalFixtureID(%s) = (%d, %s, %d), expected (%d, %s, %d)",
					tt.input, sid, cod, seq, tt.expectedSid, tt.expectedCod, tt.expectedSeq)
			}
		}
	}
}

func TestNaturalIdentityHash(t *testing.T) {
	t1 := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 9, 18, 20, 0, 0, 0, time.UTC)

	// Same team variations and identical time should yield identical hash
	h1 := NaturalIdentityHash(1, "Arsenal FC", "Chelsea FC", t1)
	h2 := NaturalIdentityHash(1, "arsenal", "chelsea", t2)
	if h1 != h2 {
		t.Errorf("expected hashes to match for normalized names: %s != %s", h1, h2)
	}

	// Different kickoff time should yield different hash
	h3 := NaturalIdentityHash(1, "Arsenal", "Chelsea", t3)
	if h1 == h3 {
		t.Errorf("different kickoff times must not collide: %s == %s", h1, h3)
	}
}

func TestNormalizeSportID_NoCollisions(t *testing.T) {
	// 1. Verify all 12 Canonical numeric IDs return their exact identity
	canonicalTests := []struct {
		input       interface{}
		expectedID  int
		expectedCod string
	}{
		{1, SportIDFootball, "FB"},
		{"1", SportIDFootball, "FB"},
		{2, SportIDBasketball, "BB"},
		{"2", SportIDBasketball, "BB"},
		{3, SportIDTennis, "TN"},
		{"3", SportIDTennis, "TN"},
		{4, SportIDIceHockey, "IH"},
		{"4", SportIDIceHockey, "IH"},
		{5, SportIDVolleyball, "VB"},
		{"5", SportIDVolleyball, "VB"}, // MUST NOT BE TENNIS!
		{6, SportIDMMA, "MM"},
		{"6", SportIDMMA, "MM"},
		{7, SportIDRugby, "RB"},
		{"7", SportIDRugby, "RB"},
		{8, SportIDCricket, "CK"},
		{"8", SportIDCricket, "CK"},
		{9, SportIDAmericanFootball, "AF"},
		{"9", SportIDAmericanFootball, "AF"},
		{10, SportIDHandball, "HB"},
		{"10", SportIDHandball, "HB"}, // MUST NOT BE MMA!
		{11, SportIDBaseball, "BS"},
		{"11", SportIDBaseball, "BS"},
		{13, SportIDTableTennis, "TT"},
		{"13", SportIDTableTennis, "TT"},
	}

	for _, tt := range canonicalTests {
		id, code, _ := NormalizeSportID(tt.input)
		if id != tt.expectedID || code != tt.expectedCod {
			t.Errorf("NormalizeSportID(%v) = (%d, %s), expected (%d, %s)", tt.input, id, code, tt.expectedID, tt.expectedCod)
		}
	}

	// 2. Verify Sportradar URNs map accurately to platform canonical IDs
	srTests := []struct {
		urn         string
		expectedID  int
		expectedCod string
	}{
		{"sr:sport:1", SportIDFootball, "FB"},
		{"sr:sport:2", SportIDBasketball, "BB"},
		{"sr:sport:3", SportIDBaseball, "BS"}, // Sportradar 3 is Baseball!
		{"sr:sport:4", SportIDIceHockey, "IH"},
		{"sr:sport:5", SportIDTennis, "TN"}, // Sportradar 5 is Tennis!
		{"sr:sport:6", SportIDHandball, "HB"},
		{"sr:sport:12", SportIDRugby, "RB"},
		{"sr:sport:16", SportIDAmericanFootball, "AF"},
		{"sr:sport:20", SportIDTableTennis, "TT"},
		{"sr:sport:21", SportIDCricket, "CK"},
		{"sr:sport:23", SportIDVolleyball, "VB"},
		{"sr:sport:117", SportIDMMA, "MM"},
	}

	for _, tt := range srTests {
		id, code, _ := NormalizeSportID(tt.urn)
		if id != tt.expectedID || code != tt.expectedCod {
			t.Errorf("NormalizeSportID(%s) = (%d, %s), expected (%d, %s)", tt.urn, id, code, tt.expectedID, tt.expectedCod)
		}
	}
}

func TestNormalizeSportIDStrict_UnknownFailsClosed(t *testing.T) {
	unknownInputs := []interface{}{
		nil,
		"",
		"   ",
		"cricket66",
		66,         // Legacy cricket MUST fail strict resolution (canonical is 8)
		"66",       // Legacy string ID 66 MUST fail
		12,         // Reserved / unused ID 12
		"12",       // Reserved / unused ID 12
		999,        // Non-existent ID
		-1,         // Negative ID
		"curling",  // Unmapped sport
		"kabaddi",  // Unmapped sport
		"sr:sport:9999",
	}

	for _, input := range unknownInputs {
		info, err := NormalizeSportIDStrict(input)
		if err == nil {
			t.Errorf("expected ErrUnknownSport for input %v, got %+v", input, info)
		}
	}
}

func TestSportReleasePolicy(t *testing.T) {
	// All 12 platform sports are platform-capable and settleable
	expectedReleased := []int{
		SportIDFootball,
		SportIDBasketball,
		SportIDTennis,
		SportIDIceHockey,
		SportIDVolleyball,
		SportIDMMA,
		SportIDRugby,
		SportIDCricket,
		SportIDAmericanFootball,
		SportIDHandball,
		SportIDBaseball,
		SportIDTableTennis,
	}

	for _, id := range expectedReleased {
		if !IsSportClientReleased(id) {
			t.Errorf("expected sport %d to be platform released", id)
		}
		if !IsSportSettleable(id) {
			t.Errorf("expected sport %d to be settleable", id)
		}
		if IsSportBlocked(id) {
			t.Errorf("sport %d should not be blocked globally", id)
		}
	}

	// Unknown sports
	for _, id := range []int{0, 12, 66, 99} {
		if IsSportClientReleased(id) {
			t.Errorf("unknown sport %d must not be released", id)
		}
	}

	// Check OfficialClientSportIDs count and order
	ids := OfficialClientSportIDs()
	if len(ids) != 12 {
		t.Fatalf("expected 12 platform sports, got %d", len(ids))
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("expected OfficialClientSportIDs to be strictly sorted: %v", ids)
		}
	}
}

func TestFixtureAliasAndProviderNormalization(t *testing.T) {
	tests := []struct {
		provider string
		rawKey   string
		expected string
	}{
		{"SPORTYBET", "sr:match:73221004", "73221004"},
		{"sportybet", "sr:match:73221004", "73221004"},
		{"ODIBETS", "73221004", "73221004"},
		{"odibets", "  73221004  ", "73221004"},
	}

	for _, tt := range tests {
		got := NormalizeProviderEventKey(tt.provider, tt.rawKey)
		if got != tt.expected {
			t.Errorf("NormalizeProviderEventKey(%q, %q) = %q, expected %q", tt.provider, tt.rawKey, got, tt.expected)
		}
	}

	alias := FixtureAlias{
		CanonicalFixtureID: "SPD-FB-00982134",
		Provider:           "SPORTYBET",
		ProviderFixtureID:  "sr:match:73221004",
		VerifiedAt:         time.Now().UTC(),
		VerificationMethod: "SR_URN_MATCH",
	}

	if alias.CanonicalFixtureID != "SPD-FB-00982134" || alias.Provider != "SPORTYBET" {
		t.Errorf("FixtureAlias properties not properly preserved: %+v", alias)
	}
}

func TestValidateFixtureLinkage(t *testing.T) {
	now := time.Now().UTC()

	sportyFix := FixtureLinkageValidation{
		SportID:          SportIDFootball,
		ProviderEventKey: "sr:match:73221004",
		ScheduledKickoff: now,
		HomeTeam:         "Arsenal FC",
		AwayTeam:         "Chelsea FC",
		CompetitionName:  "Premier League",
	}

	odiMatching := FixtureLinkageValidation{
		SportID:          SportIDFootball,
		ProviderEventKey: "73221004",
		ScheduledKickoff: now.Add(5 * time.Minute),
		HomeTeam:         "Arsenal",
		AwayTeam:         "Chelsea",
		CompetitionName:  "EPL",
	}

	ok, reason := ValidateFixtureLinkage(sportyFix, odiMatching)
	if !ok {
		t.Errorf("expected matching fixtures to link successfully, failed with: %s", reason)
	}

	// Mismatched sport
	odiWrongSport := odiMatching
	odiWrongSport.SportID = SportIDBasketball
	ok, _ = ValidateFixtureLinkage(sportyFix, odiWrongSport)
	if ok {
		t.Errorf("expected sport mismatch to fail linkage")
	}

	// Mismatched kickoff time (> 45 min)
	odiLateKickoff := odiMatching
	odiLateKickoff.ScheduledKickoff = now.Add(2 * time.Hour)
	ok, _ = ValidateFixtureLinkage(sportyFix, odiLateKickoff)
	if ok {
		t.Errorf("expected kickoff mismatch to fail linkage")
	}

	// Mismatched home team with different keys
	sportyDiffKey := sportyFix
	sportyDiffKey.ProviderEventKey = "sr:match:99999999"
	odiDiffTeam := odiMatching
	odiDiffTeam.ProviderEventKey = "11111111"
	odiDiffTeam.HomeTeam = "Liverpool"
	ok, _ = ValidateFixtureLinkage(sportyDiffKey, odiDiffTeam)
	if ok {
		t.Errorf("expected team mismatch to fail linkage when keys differ")
	}
}

func TestNaturalIdentity_DisambiguationAndCollisionResistance(t *testing.T) {
	baseTime := time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC)

	// 1. Similar Teams in Same City (Manchester United vs Manchester City)
	hashManU := NaturalCandidate{SportID: 1, HomeTeam: "Manchester United", AwayTeam: "Liverpool", ScheduledStart: baseTime}.NaturalKey()
	hashManCity := NaturalCandidate{SportID: 1, HomeTeam: "Manchester City", AwayTeam: "Liverpool", ScheduledStart: baseTime}.NaturalKey()
	if hashManU == hashManCity {
		t.Fatalf("COLLISION DETECTED: Manchester United collided with Manchester City! Hash: %s", hashManU)
	}

	// 2. Distinct Global Clubs Sharing Token (Arsenal vs Arsenal Sarandi)
	hashArsenal := NaturalCandidate{SportID: 1, HomeTeam: "Arsenal", AwayTeam: "Chelsea", ScheduledStart: baseTime}.NaturalKey()
	hashArsenalSarandi := NaturalCandidate{SportID: 1, HomeTeam: "Arsenal Sarandi", AwayTeam: "Chelsea", ScheduledStart: baseTime}.NaturalKey()
	if hashArsenal == hashArsenalSarandi {
		t.Fatalf("COLLISION DETECTED: Arsenal collided with Arsenal Sarandi! Hash: %s", hashArsenal)
	}

	// 3. Senior vs Reserve Side (Barcelona vs Barcelona B / Barcelona II)
	hashBarcaSenior := NaturalCandidate{SportID: 1, HomeTeam: "Barcelona", AwayTeam: "Girona", ScheduledStart: baseTime}.NaturalKey()
	hashBarcaB := NaturalCandidate{SportID: 1, HomeTeam: "Barcelona B", AwayTeam: "Girona", ScheduledStart: baseTime}.NaturalKey()
	hashBarcaII := NaturalCandidate{SportID: 1, HomeTeam: "Barcelona II", AwayTeam: "Girona", ScheduledStart: baseTime}.NaturalKey()
	if hashBarcaSenior == hashBarcaB {
		t.Fatalf("COLLISION DETECTED: Barcelona Senior collided with Barcelona B! Hash: %s", hashBarcaSenior)
	}
	if hashBarcaB != hashBarcaII {
		t.Errorf("Barcelona B and Barcelona II should collapse to the same RESERVE key, got %s != %s", hashBarcaB, hashBarcaII)
	}

	// 4. Senior vs Youth / Age Category (Chelsea vs Chelsea U21)
	hashChelseaSenior := NaturalCandidate{SportID: 1, HomeTeam: "Chelsea", AwayTeam: "Fulham", ScheduledStart: baseTime}.NaturalKey()
	hashChelseaU21 := NaturalCandidate{SportID: 1, HomeTeam: "Chelsea U21", AwayTeam: "Fulham", ScheduledStart: baseTime}.NaturalKey()
	if hashChelseaSenior == hashChelseaU21 {
		t.Fatalf("COLLISION DETECTED: Chelsea Senior collided with Chelsea U21! Hash: %s", hashChelseaSenior)
	}

	// 5. Men vs Women (Arsenal vs Arsenal Women)
	hashArsenalMen := NaturalCandidate{SportID: 1, HomeTeam: "Arsenal", AwayTeam: "Chelsea", ScheduledStart: baseTime, Gender: "MEN"}.NaturalKey()
	hashArsenalWomen := NaturalCandidate{SportID: 1, HomeTeam: "Arsenal Women", AwayTeam: "Chelsea Women", ScheduledStart: baseTime}.NaturalKey()
	if hashArsenalMen == hashArsenalWomen {
		t.Fatalf("COLLISION DETECTED: Arsenal Men collided with Arsenal Women! Hash: %s", hashArsenalMen)
	}

	// 6. Rescheduled Match (15:00 UTC moved to 19:00 UTC)
	rescheduledTime := baseTime.Add(4 * time.Hour)
	hashOriginal := NaturalCandidate{SportID: 1, HomeTeam: "Real Madrid", AwayTeam: "Barcelona", ScheduledStart: baseTime}.NaturalKey()
	hashRescheduled := NaturalCandidate{SportID: 1, HomeTeam: "Real Madrid", AwayTeam: "Barcelona", ScheduledStart: rescheduledTime}.NaturalKey()
	if hashOriginal == hashRescheduled {
		t.Fatalf("COLLISION DETECTED: Rescheduled match collided with original match! Hash: %s", hashOriginal)
	}

	// 7. Kickoff Boundary Jitter across 30-min window (19:29 vs 19:31)
	tJitter1 := time.Date(2026, 10, 24, 19, 29, 0, 0, time.UTC)
	tJitter2 := time.Date(2026, 10, 24, 19, 31, 0, 0, time.UTC)
	cand1 := NaturalCandidate{SportID: 1, HomeTeam: "Juventus", AwayTeam: "Milan", ScheduledStart: tJitter1}
	cand2 := NaturalCandidate{SportID: 1, HomeTeam: "Juventus", AwayTeam: "Milan", ScheduledStart: tJitter2}
	
	// Direct single-bucket hash will differ across boundary
	h1 := cand1.NaturalKey()
	h2 := cand2.NaturalKey()
	if h1 == h2 {
		t.Errorf("Expected different primary bucket hashes across boundary: %s == %s", h1, h2)
	}

	// But Candidate list (neighbouring buckets) MUST intersect, allowing cross-feed matching
	candidates1 := cand1.NaturalKeyCandidates()
	candidates2 := cand2.NaturalKeyCandidates()
	matched := false
	for _, c1 := range candidates1 {
		for _, c2 := range candidates2 {
			if c1 == c2 {
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}
	if !matched {
		t.Fatalf("FAIL: NaturalKeyCandidates failed to bridge 2-minute boundary jitter (19:29 vs 19:31)!")
	}

	// 8. Simultaneous Fixtures in Same Sport
	hashMatchA := NaturalCandidate{SportID: 1, HomeTeam: "Liverpool", AwayTeam: "Everton", ScheduledStart: baseTime}.NaturalKey()
	hashMatchB := NaturalCandidate{SportID: 1, HomeTeam: "Newcastle", AwayTeam: "Sunderland", ScheduledStart: baseTime}.NaturalKey()
	if hashMatchA == hashMatchB {
		t.Fatalf("COLLISION DETECTED: Simultaneous matches collided! Hash: %s", hashMatchA)
	}

	t.Logf("All natural identity disambiguation and collision-resistance checks PASSED.")
}


