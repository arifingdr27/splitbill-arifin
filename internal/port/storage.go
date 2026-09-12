package port

import "context"

// StorageUploader persists receipt images and returns a public or relative URL.
type StorageUploader interface {
	Upload(ctx context.Context, objectName string, data []byte, contentType string) (url string, err error)
}
