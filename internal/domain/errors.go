package domain

import "errors"

var (
	ErrImageRequired      = errors.New("image file is required")
	ErrInvalidImageType   = errors.New("image must be jpg, jpeg, or png")
	ErrPayloadTooLarge    = errors.New("payload too large")
	ErrTooManyRequests    = errors.New("too many requests")
	ErrServiceBusy        = errors.New("service temporarily unavailable")
	ErrStorageUnavailable = errors.New("storage backend is not configured")
	ErrExtractFailed      = errors.New("failed to extract receipt data")

	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidGoogleToken = errors.New("invalid google token")
	ErrQuotaExceeded    = errors.New("Free quota and credits are exhausted")
)
