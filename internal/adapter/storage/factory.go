package storage

import (
	"context"
	"fmt"

	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/sirupsen/logrus"
)

func NewUploader(ctx context.Context, cfg *config.Config, log *logrus.Logger) (port.StorageUploader, error) {
	switch cfg.BucketStorage {
	case config.StorageVM:
		if err := ensureLocalDirs(cfg.StorageLocalPath); err != nil {
			return nil, err
		}
		return NewVMStorage(cfg.StorageLocalPath, log), nil
	case config.StorageFirebase:
		return NewFirebaseStorage(ctx, cfg, log)
	default:
		return nil, fmt.Errorf("unsupported BUCKET_STORAGE: %s", cfg.BucketStorage)
	}
}
