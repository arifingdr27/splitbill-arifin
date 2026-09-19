package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/receiptprompt"
	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/sirupsen/logrus"
	"google.golang.org/genai"
)

const maxExtractAttempts = 2

type Extractor struct {
	client  *genai.Client
	model   string
	timeout time.Duration
	log     *logrus.Logger
}

func NewExtractor(ctx context.Context, apiKey, model string, timeout time.Duration, log *logrus.Logger) (*Extractor, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	if timeout < time.Second {
		timeout = 25 * time.Second
	}
	return &Extractor{client: client, model: model, timeout: timeout, log: log}, nil
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	e.log.WithFields(logrus.Fields{
		"provider": "gemini",
		"model":    e.model,
	}).Info("extract using provider")

	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	parts := []*genai.Part{
		genai.NewPartFromText(receiptprompt.Extract),
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
		attemptCtx, cancel := context.WithTimeout(ctx, e.timeout)
		result, err := e.client.Models.GenerateContent(attemptCtx, e.model, contents, cfg)
		cancel()
		if err != nil {
			e.log.Errorf("gemini attempt=%d generate failed: %v", attempt, err)
			return nil, domain.ErrExtractFailed
		}

		raw := result.Text()
		cleaned := receiptprompt.CleanJSON(raw)
		e.log.Debugf("gemini attempt=%d raw_len=%d cleaned_len=%d", attempt, len(raw), len(cleaned))

		if cleaned == "" {
			lastErr = fmt.Errorf("empty response")
			e.log.Warnf("gemini attempt=%d empty cleaned response", attempt)
			continue
		}

		var out domain.SplitbillResult
		if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
			lastErr = err
			e.log.Errorf("gemini attempt=%d unmarshal failed cleaned=%q err=%v", attempt, receiptprompt.Truncate(cleaned, 500), err)
			continue
		}
		return &out, nil
	}

	e.log.Errorf("gemini extract failed after retries: %v", lastErr)
	return nil, domain.ErrExtractFailed
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
	currency := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"code":       str,
			"symbol":     str,
			"name":       str,
			"confidence": str,
		},
		Required: []string{"code", "symbol", "name", "confidence"},
	}
	language := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"code":       str,
			"name":       str,
			"confidence": str,
		},
		Required: []string{"code", "name", "confidence"},
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
			"currency": currency,
			"language": language,
		},
		Required: []string{"items", "store_information", "totals", "transaction_information", "currency", "language"},
	}
}
