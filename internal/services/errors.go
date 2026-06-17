package services

import "errors"

// Standard service errors for comprehensive error handling testing
var (
	// Network and connectivity errors
	ErrNetworkUnavailable = errors.New("network unavailable")
	ErrTimeout            = errors.New("operation timed out")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrForbidden          = errors.New("access forbidden")

	// Data errors
	ErrNotFound      = errors.New("resource not found")
	ErrInvalidInput  = errors.New("invalid input provided")
	ErrInvalidFormat = errors.New("invalid format")
	ErrDataCorrupted = errors.New("data corrupted")

	// Cache errors
	ErrCacheUnavailable = errors.New("cache unavailable")
	ErrCacheMiss        = errors.New("cache miss")
	ErrCacheCorrupted   = errors.New("cache corrupted")

	// Service errors
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrQuotaExceeded      = errors.New("quota exceeded")
	ErrRateLimited        = errors.New("rate limited")

	// AI service specific errors
	ErrAIServiceDown   = errors.New("AI service down")
	ErrContextTooLarge = errors.New("context too large")
	ErrInvalidPrompt   = errors.New("invalid prompt")

	// Email service specific errors
	ErrMessageNotFound  = errors.New("message not found")
	ErrLabelNotFound    = errors.New("label not found")
	ErrInvalidMessageID = errors.New("invalid message ID")
	ErrInvalidLabelID   = errors.New("invalid label ID")
)
