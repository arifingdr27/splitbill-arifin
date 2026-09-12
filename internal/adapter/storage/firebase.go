package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

type FirebaseStorage struct {
	bucket *storage.BucketHandle
	log    *logrus.Logger
}

func NewFirebaseStorage(ctx context.Context, cfg *config.Config, log *logrus.Logger) (*FirebaseStorage, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.FirebaseCredPath))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}

	client, err := app.Storage(ctx)
	if err != nil {
		return nil, fmt.Errorf("init firebase storage: %w", err)
	}

	bucket, err := client.Bucket(cfg.FirebaseBucket)
	if err != nil {
		return nil, fmt.Errorf("get firebase bucket: %w", err)
	}

	log.Info("Firebase storage initialized")
	return &FirebaseStorage{bucket: bucket, log: log}, nil
}

func (s *FirebaseStorage) Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	reader, err := prepareReader(data, s.log)
	if err != nil {
		return "", err
	}

	wc := s.bucket.Object(objectName).NewWriter(ctx)
	wc.ContentType = contentType

	if _, err := io.Copy(wc, reader); err != nil {
		_ = wc.Close()
		return "", fmt.Errorf("upload to firebase: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("close firebase writer: %w", err)
	}

	attrs, err := s.bucket.Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("get bucket attrs: %w", err)
	}

	publicURL := fmt.Sprintf(
		"https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media",
		attrs.Name,
		url.PathEscape(objectName),
	)
	s.log.Infof("Firebase upload successful: %s", publicURL)
	return publicURL, nil
}
