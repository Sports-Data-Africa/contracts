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
		{"Football Over/Under Goals", 1, "OVER_UNDER", "Total Goals Over/Under", true},
		{"Football BTTS", 1, "BTTS", "Both Teams To Score", true},
		{"Football Double Chance", 1, "DOUBLE_CHANCE", "Double Chance", true},
		{"Football Draw No Bet", 1, "DRAW_NO_BET", "Draw No Bet", true},
		{"Football Asian Handicap", 1, "FB_ASIAN_HANDICAP", "Asian Handicap", true},

		// Basketball (Allowlisted incl. OT)
		{"Basketball Moneyline", 2, "MONEYLINE", "Winner (incl. overtime)", true},
		{"Basketball Total Points", 2, "TOTAL_POINTS", "Total Points (incl. overtime)", true},
		{"Basketball Spread", 2, "SPREAD", "Spread (incl. overtime)", true},

		// Ice Hockey (Allowlisted Moneyline only)
		{"Ice Hockey Moneyline", 4, "MONEYLINE", "Winner (incl. overtime)", true},

		// Tennis, Volleyball, Rugby, Cricket, MMA
		{"Tennis Match Winner", 3, "MATCH_WINNER", "Match Winner", true},
		{"Volleyball Match Winner", 5, "MATCH_WINNER", "Match Winner", true},
		{"Rugby 1X2", 7, "1X2", "Match Winner", true},
		{"Cricket Match Winner", 8, "MATCH_WINNER", "Match Winner", true},
		{"MMA Fight Winner", 6, "FIGHT_WINNER", "Fight Winner", true},

		// Blocked / Suspended Markets
		{"Football Corners Over/Under", 1, "FB_CORNERS_TOTAL", "Corners Over/Under 9.5", false},
		{"Football Cards Over/Under", 1, "FB_CARDS_TOTAL", "Cards Over/Under 3.5", false},
		{"Football 1st Half 1X2", 1, "FIRST_HALF_1X2", "1st Half 1X2", false},
		{"Basketball 1st Quarter Winner", 2, "Q1_WINNER", "1st Quarter Winner", false},
		{"Ice Hockey Regulation 1X2", 4, "1X2", "Regulation 1X2 (3-Way)", false},
		{"Table Tennis Match Winner", 13, "TT_MATCH_WINNER", "Match Winner", false},
		{"MMA Method of Victory", 6, "MM_METHOD_OF_VICTORY", "Method of Victory", false},
		{"Rugby Tries", 7, "TOTAL_TRIES", "Total Tries Over/Under", false},
		{"Cricket Most Sixes", 8, "MOST_SIXES", "Most Sixes", false},
		{"Tennis Total Games", 3, "TOTAL_GAMES", "Total Games Over/Under", false},
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

func TestIsMarketSettleableAndAllowlisted_DisabledKillswitch(t *testing.T) {
	_ = os.Setenv("BETTING_ALLOWLIST_ENABLED", "false")
	defer os.Unsetenv("BETTING_ALLOWLIST_ENABLED")

	allowed, _ := IsMarketSettleableAndAllowlisted(13, "TT_MATCH_WINNER", "Match Winner")
	if !allowed {
		t.Errorf("expected allowed=true when BETTING_ALLOWLIST_ENABLED=false")
	}
}
