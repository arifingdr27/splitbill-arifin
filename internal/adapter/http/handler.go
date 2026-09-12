package httpadapter

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type SplitbillHandler struct {
	svc            *service.SplitbillService
	quota          *service.QuotaService
	log            *logrus.Logger
	maxUploadBytes int
}

func NewSplitbillHandler(svc *service.SplitbillService, quota *service.QuotaService, log *logrus.Logger, cfg *config.Config) *SplitbillHandler {
	return &SplitbillHandler{
		svc:            svc,
		quota:          quota,
		log:            log,
		maxUploadBytes: cfg.HTTPBodyLimitBytes,
	}
}

// Extract processes receipt image and extracts splitbill information
// @Summary Extract splitbill information from receipt image
// @Description Upload a receipt image and extract detailed splitbill information including items, store details, totals, and transaction information using OCR and AI
// @Tags Splitbill
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Receipt image file (jpg, jpeg, png)"
// @Success 200 {object} domain.SplitbillResult "Successfully processed receipt"
// @Failure 400 {object} domain.ErrorResponse "Invalid request"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 402 {object} domain.QuotaExceededResponse "Quota exceeded"
// @Failure 413 {object} domain.ErrorResponse "Payload too large"
// @Failure 422 {object} domain.ErrorResponse "Failed to process receipt"
// @Failure 429 {object} domain.ErrorResponse "Too many requests"
// @Router /api/v2 [post]
func (h *SplitbillHandler) Extract(c *fiber.Ctx) error {
	userID, err := UserIDFromCtx(c)
	if err != nil {
		return writeError(c, err)
	}

	if _, err := h.quota.AssertAvailable(c.UserContext(), userID); err != nil {
		if errors.Is(err, domain.ErrQuotaExceeded) {
			snap, _ := h.quota.Snapshot(c.UserContext(), userID)
			return writeQuotaExceeded(c, snap)
		}
		h.log.WithError(err).Error("quota assert failed")
		return writeError(c, err)
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		h.log.Warnf("missing image form field: %v", err)
		return writeError(c, domain.ErrImageRequired)
	}

	if fileHeader.Size > int64(h.maxUploadBytes) {
		h.log.WithField("size", fileHeader.Size).Warn("upload rejected: oversized")
		return writeError(c, domain.ErrPayloadTooLarge)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return writeError(c, domain.ErrImageRequired)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, int64(h.maxUploadBytes)+1))
	if err != nil {
		h.log.Errorf("read upload failed: %v", err)
		return writeError(c, domain.ErrImageRequired)
	}
	if len(data) > h.maxUploadBytes {
		h.log.WithField("size", len(data)).Warn("upload rejected: oversized")
		return writeError(c, domain.ErrPayloadTooLarge)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mimeFromExt(fileHeader.Filename)
	}

	result, err := h.svc.ExtractReceipt(c.UserContext(), domain.ImageInput{
		Filename:    fileHeader.Filename,
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		h.log.Errorf("extract receipt failed: %v", err)
		return writeError(c, err)
	}

	if _, err := h.quota.DebitOnSuccess(c.UserContext(), userID); err != nil {
		h.log.WithError(err).WithField("user_id", userID.String()).Error("quota debit after OCR success failed")
	}
	return writeSuccess(c, result)
}

func FiberErrorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) && fe.Code == fiber.StatusRequestEntityTooLarge {
		return writeError(c, domain.ErrPayloadTooLarge)
	}
	return writeError(c, err)
}

func mimeFromExt(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}
