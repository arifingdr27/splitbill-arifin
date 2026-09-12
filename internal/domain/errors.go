package domain

import "errors"

var (
	ErrImageRequired      = errors.New("image file is required")
	ErrInvalidImageType   = errors.New("image must be jpg, jpeg, or png")
	ErrStorageUnavailable = errors.New("storage backend is not configured")
	ErrExtractFailed      = errors.New("failed to extract receipt data")
)
