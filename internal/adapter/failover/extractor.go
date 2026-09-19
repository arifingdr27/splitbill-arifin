package failover

import (
	"context"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/sirupsen/logrus"
)

const hedgeAfter = 8 * time.Second

// Extractor tries primary, hedges fallback after hedgeAfter, or on primary failure.
type Extractor struct {
	primary  port.ReceiptExtractor
	fallback port.ReceiptExtractor
	log      *logrus.Logger
}

func New(primary, fallback port.ReceiptExtractor, log *logrus.Logger) *Extractor {
	return &Extractor{primary: primary, fallback: fallback, log: log}
}

type outcome struct {
	result *domain.SplitbillResult
	err    error
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	if e.fallback == nil {
		return e.primary.Extract(ctx, image, mimeType)
	}

	run := func(parent context.Context, ex port.ReceiptExtractor) (context.CancelFunc, <-chan outcome) {
		cctx, cancel := context.WithCancel(parent)
		ch := make(chan outcome, 1)
		go func() {
			res, err := ex.Extract(cctx, image, mimeType)
			ch <- outcome{result: res, err: err}
		}()
		return cancel, ch
	}

	primaryCancel, primaryCh := run(ctx, e.primary)
	defer primaryCancel()

	hedgeTimer := time.NewTimer(hedgeAfter)
	defer hedgeTimer.Stop()

	var (
		fallbackCancel context.CancelFunc
		fallbackCh     <-chan outcome
		primaryDone    bool
		fallbackDone   bool
		fallbackErr    error
	)
	defer func() {
		if fallbackCancel != nil {
			fallbackCancel()
		}
	}()

	startFallback := func(hedged bool) {
		if fallbackCh != nil || fallbackDone {
			return
		}
		if hedged {
			e.log.Warn("primary extract slow; starting hedged fallback")
		}
		fallbackCancel, fallbackCh = run(ctx, e.fallback)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, domain.ErrExtractFailed

		case out := <-primaryCh:
			primaryCh = nil
			primaryDone = true
			if out.err == nil {
				return out.result, nil
			}
			e.log.WithError(out.err).Warn("primary extractor failed; trying fallback")
			if fallbackDone {
				return nil, domain.ErrExtractFailed
			}
			startFallback(false)

		case <-hedgeTimer.C:
			if !primaryDone {
				startFallback(true)
			}

		case out := <-fallbackCh:
			fallbackCh = nil
			fallbackDone = true
			if out.err == nil {
				primaryCancel()
				e.log.Info("extract completed via fallback provider")
				return out.result, nil
			}
			fallbackErr = out.err
			e.log.WithError(fallbackErr).Error("fallback extractor failed")
			if primaryDone {
				return nil, domain.ErrExtractFailed
			}
		}
	}
}
