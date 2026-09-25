package errors

// Public B2B Error Codes (SDA-Native, 100% Sanitized)
const (
	ErrCodeMarketUnavailable     = "MARKET_TEMPORARILY_UNAVAILABLE"
	ErrCodeFixtureUnavailable    = "FIXTURE_UNAVAILABLE"
	ErrCodeUnderageProhibited    = "ERR_UNDERAGE_PROHIBITED"
	ErrCodeStaleOdds             = "ERR_STALE_ODDS"
	ErrCodeMarketSuspended       = "MARKET_SUSPENDED"
	ErrCodeVersionMismatch       = "ERR_VERSION_MISMATCH"
	ErrCodeUnauthorized          = "UNAUTHORIZED"
	ErrCodeRateLimitExceeded     = "RATE_LIMIT_EXCEEDED"
	ErrCodeSportNotSupported     = "ERR_SPORT_NOT_SUPPORTED"
	ErrCodeInvalidSelection      = "ERR_INVALID_SELECTION"
	ErrCodeBetRejected           = "BET_REJECTED"
)

// SanitizedClientError represents a structured, client-safe error response.
type SanitizedClientError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewSanitizedError(code, message string) SanitizedClientError {
	return SanitizedClientError{
		Code:    code,
		Message: message,
	}
}

// Pre-defined sanitized errors
var (
	ErrMarketUnavailable  = NewSanitizedError(ErrCodeMarketUnavailable, "The requested market is temporarily unavailable.")
	ErrFixtureUnavailable = NewSanitizedError(ErrCodeFixtureUnavailable, "The requested fixture is currently unavailable.")
	ErrUnderageProhibited = NewSanitizedError(ErrCodeUnderageProhibited, "Wagering on underage and minor competitions is strictly prohibited by law.")
	ErrStaleOdds          = NewSanitizedError(ErrCodeStaleOdds, "Accepted odds have expired or market version has advanced.")
	ErrMarketSuspended    = NewSanitizedError(ErrCodeMarketSuspended, "The market is currently suspended.")
	ErrVersionMismatch    = NewSanitizedError(ErrCodeVersionMismatch, "Submitted market version is outdated.")
	ErrSportNotSupported  = NewSanitizedError(ErrCodeSportNotSupported, "Sport is not available for commercial B2B wagering.")
)
