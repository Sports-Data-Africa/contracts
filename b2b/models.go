package b2b

import (
	"errors"
	"time"

	"github.com/Sports-Data-Africa/contracts/events"
)

// Allowed B2B Sports (Strict 8-Sport Client Whitelist)
const (
	SportFootball   = 1
	SportIceHockey  = 2
	SportBasketball = 3
	SportTennis     = 4
	SportVolleyball = 6
	SportRugby      = 7
	SportMMA        = 9
	SportCricket    = 66

	// Internal/Staging Only
	SportTableTennis = 10
)

var allowedB2BSports = map[int]string{
	SportFootball:   "Football",
	SportIceHockey:  "Ice Hockey",
	SportBasketball: "Basketball",
	SportTennis:     "Tennis",
	SportVolleyball: "Volleyball",
	SportRugby:      "Rugby",
	SportMMA:        "MMA",
	SportCricket:    "Cricket",
}

// IsSportAllowedForB2B checks if a sport is eligible for B2B client exposure.
// Sport 10 (Table Tennis) returns false — internal only.
func IsSportAllowedForB2B(sportID int) bool {
	_, ok := allowedB2BSports[sportID]
	return ok
}

var ErrSportNotEligibleForB2B = errors.New("ERR_SPORT_NOT_ELIGIBLE_FOR_B2B")

// B2BMarketOddsUpdate is serialized strictly for external clients over WebSocket, SSE, and REST.
// ZERO upstream provider fields allowed.
type B2BMarketOddsUpdate struct {
	Event     string    `json:"event"`      // "MARKET_ODDS_UPDATED"
	FixtureID int64     `json:"fixture_id"` // SDA Canonical ID (e.g. 88001001)
	SportID   int       `json:"sport_id"`
	Market    B2BMarket `json:"market"`
	Timestamp int64     `json:"timestamp"`
}

type B2BMarket struct {
	Code       string         `json:"code"`       // Canonical market code (e.g. "1X2")
	Version    int64          `json:"version"`    // Sovereign monotonic market version
	Suspended  bool           `json:"suspended"`  // Suspension status
	Selections []B2BSelection `json:"selections"`
}

type B2BSelection struct {
	Code string  `json:"code"` // "HOME", "DRAW", "AWAY", "OVER", "UNDER"
	Odds float64 `json:"odds"` // Decimal odds (e.g. 1.95)
}

// B2BFixtureSnapshot is the public representation of a fixture.
type B2BFixtureSnapshot struct {
	FixtureID      int64     `json:"fixture_id"`      // SDA Canonical Fixture ID
	CompetitionID  int64     `json:"competition_id"`  // SDA Canonical Competition ID (>= 100001)
	SportID        int       `json:"sport_id"`
	HomeTeam       string    `json:"home_team"`
	AwayTeam       string    `json:"away_team"`
	StartTime      time.Time `json:"start_time"`
	Status         string    `json:"status"`          // "PREMATCH", "LIVE", "FINISHED"
	ScoreHome      int       `json:"score_home"`
	ScoreAway      int       `json:"score_away"`
	PeriodName     string    `json:"period_name,omitempty"`
	ElapsedMinutes int       `json:"elapsed_minutes,omitempty"`
}

// B2BTicketResponse is returned to B2B clients upon bet placement.
// Placed provider is kept strictly internal.
type B2BTicketResponse struct {
	TicketID      string  `json:"ticket_id"`
	FixtureID     int64   `json:"fixture_id"`
	MarketCode    string  `json:"market_code"`
	SelectionCode string  `json:"selection_code"`
	AcceptedOdds  float64 `json:"accepted_odds"`
	MarketVersion int64   `json:"market_version"`
	Status        string  `json:"status"` // "ACCEPTED", "REJECTED"
	ErrorMessage  string  `json:"error_message,omitempty"`
}

// B2BSettlementEvent is the public settlement webhook payload delivered to clients.
// Cryptographic HMAC-SHA256 signature is delivered in headers (X-SDA-Signature).
type B2BSettlementEvent struct {
	Event     string            `json:"event"`      // "BET_SETTLED"
	FixtureID int64             `json:"fixture_id"` // SDA Canonical ID
	TicketID  string            `json:"ticket_id"`  // Client ticket reference
	Result    B2BMatchResult    `json:"result"`
	Status    string            `json:"status"`     // "WON", "LOST", "VOID"
	Payout    float64           `json:"payout"`
	SettledAt int64             `json:"settled_at"` // Unix epoch timestamp
}

type B2BMatchResult struct {
	HomeScore int `json:"home_score"`
	AwayScore int `json:"away_score"`
}

// Strict Transformer: Converts InternalMarketOddsUpdatedEvent -> B2BMarketOddsUpdate
// Drops all commercial provider metadata and upstream identifiers.
func ToB2BMarketOddsUpdate(in *events.InternalMarketOddsUpdatedEvent) (*B2BMarketOddsUpdate, error) {
	if !IsSportAllowedForB2B(in.SportID) {
		return nil, ErrSportNotEligibleForB2B
	}

	selections := make([]B2BSelection, len(in.Selections))
	for i, s := range in.Selections {
		selections[i] = B2BSelection{
			Code: s.Code,
			Odds: s.Odds,
		}
	}

	return &B2BMarketOddsUpdate{
		Event:     "MARKET_ODDS_UPDATED",
		FixtureID: in.FixtureID,
		SportID:   in.SportID,
		Market: B2BMarket{
			Code:       in.MarketCode,
			Version:    in.MarketVersion,
			Suspended:  in.IsSuspended,
			Selections: selections,
		},
		Timestamp: in.PublishedAt,
	}, nil
}

// Strict Transformer: Converts InternalResultCertifiedEvent -> B2BSettlementEvent
// Drops verification sources and evidence hashes from client payload.
func ToB2BSettlementEvent(in *events.InternalResultCertifiedEvent, ticketID string, status string, payout float64) (*B2BSettlementEvent, error) {
	if !IsSportAllowedForB2B(in.SportID) {
		return nil, ErrSportNotEligibleForB2B
	}

	return &B2BSettlementEvent{
		Event:     "BET_SETTLED",
		FixtureID: in.FixtureID,
		TicketID:  ticketID,
		Result: B2BMatchResult{
			HomeScore: in.HomeScore,
			AwayScore: in.AwayScore,
		},
		Status:    status,
		Payout:    payout,
		SettledAt: in.VerifiedAt.Unix(),
	}, nil
}
