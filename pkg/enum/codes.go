package enum

// ErrorCode represents an error code as a string.
type ErrorCode string

// Error codes constants, unexported.
const (
	Unauthorized        ErrorCode = "UNAUTHORIZED"
	ApiError            ErrorCode = "API_ERROR"
	NotAllowed          ErrorCode = "NOT_ALLOWED"
	NotFound            ErrorCode = "NOT_FOUND"
	AlreadyExists       ErrorCode = "ALREADY_EXISTS"
	WrongCredentials    ErrorCode = "WRONG_CREDENTIALS"
	InvalidTurnstile    ErrorCode = "INVALID_TURNSTILE"
	MalformedRequest    ErrorCode = "MALFORMED_REQUEST"
	InvalidCookie       ErrorCode = "INVALID_COOKIE"
	InvalidAccessToken  ErrorCode = "INVALID_ACCESS_TOKEN"
	InvalidRefreshToken ErrorCode = "INVALID_REFRESH_TOKEN"
	ExpiredAccessToken  ErrorCode = "EXPIRED_ACCESS_TOKEN"
	ExpiredRefreshToken ErrorCode = "EXPIRED_REFRESH_TOKEN"
	UserNotFound        ErrorCode = "USER_NOT_FOUND"
)
