package market

import (
	"os"
	"testing"
)

func TestIsMarketSettleableAndAllowlisted_Matrix(t *testing.T) {
	tests := []struct {
		name        string
		sportID     int
		marketCode  string
		marketName  string
		expectAllow bool
	}{
		// Football (Allowlisted)
		{"Football 1X2", 1, "1X2", "Match Result", true},
		{"Football Over/Under Goals", 1, "OVER_UNDER", "Total Goals Over/Under 2.5", true},
		{"Football BTTS", 1, "BTTS", "Both Teams To Score", true},
		{"Football Double Chance", 1, "DOUBLE_CHANCE", "Double Chance", true},
		{"Football Draw No Bet", 1, "DRAW_NO_BET", "Draw No Bet", true},
		{"Football Asian Handicap", 1, "FB_ASIAN_HANDICAP", "Asian Handicap -0.5", true},

		// Basketball (Allowlisted incl. OT)
		{"Basketball Moneyline", 2, "MONEYLINE", "Winner (incl. overtime)", true},
		{"Basketball Total Points", 2, "TOTAL_POINTS", "Total Points Over/Under 215.5", true},
		{"Basketball Spread", 2, "SPREAD", "Spread -4.5 (incl. overtime)", true},

		// Ice Hockey (Allowlisted Moneyline only)
		{"Ice Hockey Moneyline", 4, "MONEYLINE", "Winner (incl. overtime)", true},

		// Tennis, Volleyball, Rugby, Cricket, MMA
		{"Tennis Match Winner", 3, "MATCH_WINNER", "Match Winner", true},
		{"Volleyball Match Winner", 5, "MATCH_WINNER", "Match Winner", true},
		{"Rugby 1X2", 7, "1X2", "Match Winner", true},
		{"Cricket Match Winner", 8, "MATCH_WINNER", "Match Winner", true},
		{"MMA Fight Winner", 6, "FIGHT_WINNER", "Fight Winner", true},

		// Football 1st Half (Verified & Allowlisted)
		{"Football 1st Half 1X2", 1, "FIRST_HALF_1X2", "1st Half 1X2", true},
		{"Football 1st Half Over/Under", 1, "1st Half - Over/Under", "1st Half - Over/Under 0.5", true},

		// Operation Full Activation — Certified and Released Markets
		{"Football Corners Over/Under", 1, "FB_CORNERS_TOTAL", "Corners Over/Under 9.5", true},
		{"Football Cards Over/Under", 1, "FB_CARDS_TOTAL", "Cards Over/Under 3.5", true},
		{"Football 2nd Half 1X2", 1, "2ND_HALF_1X2", "2nd Half 1X2", true},
		{"Basketball 1st Quarter Winner", 2, "Q1_WINNER", "1st Quarter Winner", true},
		{"Ice Hockey Regulation 1X2", 4, "1X2", "Regulation 1X2 (3-Way)", true},
		{"Table Tennis Match Winner", 13, "TT_MATCH_WINNER", "Match Winner", true},
		{"MMA Method of Victory", 6, "MM_METHOD_OF_VICTORY", "Method of Victory", true},
		{"Rugby Tries", 7, "TOTAL_TRIES", "Total Tries Over/Under 4.5", true},
		{"Cricket Most Sixes", 8, "MOST_SIXES", "Most Sixes", true},
		{"Tennis Total Games", 3, "TOTAL_GAMES", "Total Games Over/Under 21.5", true},
		{"American Football Released", 9, "AF_MONEYLINE", "Moneyline", true},
		{"Handball Released", 10, "HB_1X2", "Match Result", true},
		{"Baseball Released", 11, "BS_MONEYLINE", "Moneyline", true},

		// Strictly Reject Unknown / Unmapped Fuzzy Markets (No broad regex pass)
		{"Unmapped Throw-Ins (Contains TOTAL)", 1, "TOTAL_THROW_INS", "Total Throw-Ins Over/Under", false},
		{"Unmapped Goal Prop (Contains GOAL)", 1, "NEXT_GOALSCORER", "Next Goalscorer", false},
		{"Unmapped Time Winner (Contains WINNER)", 1, "MINUTE_15_WINNER", "15 Minute Winner", false},
		{"Unmapped Prop Handicap (Contains HANDICAP)", 1, "CORNER_HANDICAP_PROP", "Corner Handicap Match Bet", false},
		{"Unknown Sport Quarantined", 99, "XX_MONEYLINE", "Moneyline", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allowed, reason := IsMarketSettleableAndAllowlisted(tc.sportID, tc.marketCode, tc.marketName)
			if allowed != tc.expectAllow {
				t.Errorf("%s: expected allowed=%v, got %v (reason: %s)", tc.name, tc.expectAllow, allowed, reason)
			}
		})
	}
}

func TestCanPlaceMarket_DirectCapability(t *testing.T) {
	// 1. Approved full-time market
	allowed, reason := CanPlaceMarket(1, "FB_1X2", "1X2 / Match Result", PeriodFullTime)
	if !allowed {
		t.Errorf("expected FB_1X2 to be allowed for placement, got false: %s", reason)
	}

	// 2. Approved verified football half-time market
	allowedHT, reasonHT := CanPlaceMarket(1, "1ST_HALF_OVER_UNDER", "1st Half - Over/Under 0.5", PeriodHalfTime)
	if !allowedHT {
		t.Errorf("expected 1ST_HALF_OVER_UNDER in HT to be allowed, got false: %s", reasonHT)
	}

	// 3. Certified second half market
	allowed2H, reason2H := CanPlaceMarket(1, "FB_2H_1X2", "2nd Half 1X2", PeriodSecondHalf)
	if !allowed2H {
		t.Errorf("expected FB_2H_1X2 to be allowed under Full Activation, got false: %s", reason2H)
	}

	// 4. Unsupported period fails closed (e.g. Q1 on Football)
	allowedBadPeriod, _ := CanPlaceMarket(1, "FB_1X2", "Match Winner", PeriodQuarter1)
	if allowedBadPeriod {
		t.Errorf("expected unsupported period Q1 on Football to fail closed, got allowed=true")
	}

	// 5. Unmapped novelty market fails closed
	allowedNovelty, _ := CanPlaceMarket(1, "YELLOW_CARD_MINUTE", "First Yellow Card Minute", PeriodFullTime)
	if allowedNovelty {
		t.Errorf("expected unmapped novelty market to fail closed, got allowed=true")
	}
}

func TestIsMarketSettleableAndAllowlisted_DisabledKillswitch(t *testing.T) {
	_ = os.Setenv("BETTING_ALLOWLIST_ENABLED", "false")
	defer os.Unsetenv("BETTING_ALLOWLIST_ENABLED")

	allowed, _ := IsMarketSettleableAndAllowlisted(13, "TT_MATCH_WINNER", "Match Winner")
	if !allowed {
		t.Errorf("expected allowed=true when BETTING_ALLOWLIST_ENABLED=false")
	}
}
