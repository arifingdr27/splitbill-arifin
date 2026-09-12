package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/sirupsen/logrus"
)

type SplitbillService struct {
	storage   port.StorageUploader
	extractor port.ReceiptExtractor
	log       *logrus.Logger
}

func NewSplitbillService(storage port.StorageUploader, extractor port.ReceiptExtractor, log *logrus.Logger) *SplitbillService {
	return &SplitbillService{storage: storage, extractor: extractor, log: log}
}

func (s *SplitbillService) ExtractReceipt(ctx context.Context, input domain.ImageInput) (*domain.SplitbillResult, error) {
	if len(input.Data) == 0 {
		return nil, domain.ErrImageRequired
	}
	if !isAllowedImage(input.Filename, input.ContentType) {
		return nil, domain.ErrInvalidImageType
	}

	objectName := buildObjectName(input.Filename)
	url, err := s.storage.Upload(ctx, objectName, input.Data, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}
	s.log.Infof("uploaded image url=%s", url)

	result, err := s.extractor.Extract(ctx, input.Data, input.ContentType)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func buildObjectName(filename string) string {
	safe := strings.ReplaceAll(filename, " ", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	if safe == "" {
		safe = "receipt.jpg"
	}
	ts := time.Now().Format("20060102150405")
	return filepath.ToSlash(filepath.Join("images", fmt.Sprintf("%s_%s", ts, safe)))
}

func isAllowedImage(filename, contentType string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png":
		return true
	}
	ct := strings.ToLower(contentType)
	switch ct {
	case "image/jpeg", "image/jpg", "image/png":
		return true
	}
	return false
}
