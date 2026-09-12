package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/sirupsen/logrus"
	"google.golang.org/genai"
)

const maxExtractAttempts = 2

const extractPrompt = `Lakukan Optical Character Recognition (OCR) pada gambar struk ini dan ekstrak informasi belanja.
Kembalikan HANYA JSON valid sesuai schema, tanpa markdown, tanpa penjelasan, tanpa teks di luar JSON.

Aturan field:
- items: daftar barang dari struk
- price: harga per unit; jika tidak ada kolom terpisah, hitung total/quantity; jika tidak bisa, "0"
- quantity, total: sesuai struk
- nilai numerik: desimal tanpa pemisah ribuan (contoh "220000.00")
- field tidak ditemukan: string kosong ""
- date: DD/MM/YYYY
- time: HH:MM
- discount: angka desimal; jika tidak ada, "0"`

type Extractor struct {
	client *genai.Client
	model  string
	log    *logrus.Logger
}

func NewExtractor(ctx context.Context, apiKey, model string, log *logrus.Logger) (*Extractor, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &Extractor{client: client, model: model, log: log}, nil
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	parts := []*genai.Part{
		genai.NewPartFromText(extractPrompt),
		{InlineData: &genai.Blob{MIMEType: mimeType, Data: image}},
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   receiptSchema(),
	}

	var lastErr error
	for attempt := 1; attempt <= maxExtractAttempts; attempt++ {
		result, err := e.client.Models.GenerateContent(ctx, e.model, contents, cfg)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrExtractFailed, err)
		}

		raw := result.Text()
		cleaned := cleanJSON(raw)
		e.log.Debugf("gemini attempt=%d raw_len=%d cleaned_len=%d", attempt, len(raw), len(cleaned))

		if cleaned == "" {
			lastErr = fmt.Errorf("empty response")
			e.log.Warnf("gemini attempt=%d empty cleaned response", attempt)
			continue
		}

		var out domain.SplitbillResult
		if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
			lastErr = err
			e.log.Errorf("gemini attempt=%d unmarshal failed cleaned=%q err=%v", attempt, truncate(cleaned, 500), err)
			continue
		}
		return &out, nil
	}

	return nil, fmt.Errorf("%w: unmarshal: %v", domain.ErrExtractFailed, lastErr)
}

func receiptSchema() *genai.Schema {
	str := &genai.Schema{Type: genai.TypeString}
	item := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"name":     str,
			"price":    str,
			"quantity": str,
			"total":    str,
		},
		Required: []string{"name", "price", "quantity", "total"},
	}
	tax := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"amount":         str,
			"service_charge": str,
			"dpp":            str,
			"name":           str,
			"total_tax":      str,
		},
		Required: []string{"amount", "service_charge", "dpp", "name", "total_tax"},
	}
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"items": {
				Type:  genai.TypeArray,
				Items: item,
			},
			"store_information": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"address":      str,
					"email":        str,
					"npwp":         str,
					"phone_number": str,
					"store_name":   str,
				},
				Required: []string{"address", "email", "npwp", "phone_number", "store_name"},
			},
			"totals": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"change":   str,
					"discount": str,
					"payment":  str,
					"subtotal": str,
					"tax":      tax,
					"total":    str,
				},
				Required: []string{"change", "discount", "payment", "subtotal", "tax", "total"},
			},
			"transaction_information": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"date":           str,
					"time":           str,
					"transaction_id": str,
				},
				Required: []string{"date", "time", "transaction_id"},
			},
		},
		Required: []string{"items", "store_information", "totals", "transaction_information"},
	}
}

func cleanJSON(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "\uFEFF")

	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
