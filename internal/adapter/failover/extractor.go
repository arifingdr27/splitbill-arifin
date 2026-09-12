package failover

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/sirupsen/logrus"
)

// Extractor tries primary, then optional fallback on failure.
type Extractor struct {
	primary  port.ReceiptExtractor
	fallback port.ReceiptExtractor
	log      *logrus.Logger
}

func New(primary, fallback port.ReceiptExtractor, log *logrus.Logger) *Extractor {
	return &Extractor{primary: primary, fallback: fallback, log: log}
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	result, err := e.primary.Extract(ctx, image, mimeType)
	if err == nil {
		return result, nil
	}
	if e.fallback == nil {
		return nil, err
	}

	e.log.WithError(err).Warn("primary extractor failed; trying fallback")
	fb, fbErr := e.fallback.Extract(ctx, image, mimeType)
	if fbErr != nil {
		e.log.WithError(fbErr).Error("fallback extractor failed")
		return nil, domain.ErrExtractFailed
	}
	e.log.Info("extract completed via fallback provider")
	return fb, nil
}
