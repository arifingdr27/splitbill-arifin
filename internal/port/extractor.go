package port

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
)

// ReceiptExtractor extracts structured splitbill data from a receipt image.
type ReceiptExtractor interface {
	Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error)
}
