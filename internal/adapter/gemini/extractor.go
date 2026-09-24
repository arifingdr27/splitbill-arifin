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

	temp := float32(0.1)
	thinkingBudget := int32(0)
	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   receiptSchema(),
		Temperature:      &temp,
		MaxOutputTokens:  4096,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &thinkingBudget,
		},
	}

	var lastErr error
	for attempt := 1; attempt <= maxExtractAttempts; attempt++ {
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, e.timeout)
		result, err := e.client.Models.GenerateContent(attemptCtx, e.model, contents, cfg)
		cancel()
		elapsed := time.Since(started)
		if err != nil {
			e.log.Errorf("gemini attempt=%d generate failed duration=%s: %v", attempt, elapsed, err)
			return nil, domain.ErrExtractFailed
		}

		raw := result.Text()
		cleaned := receiptprompt.CleanJSON(raw)
		e.log.Infof("gemini attempt=%d ok duration=%s raw_len=%d cleaned_len=%d", attempt, elapsed, len(raw), len(cleaned))

		if cleaned == "" {
			lastErr = fmt.Errorf("empty response")
			e.log.Warnf("gemini attempt=%d empty cleaned response duration=%s", attempt, elapsed)
			continue
		}

		var out domain.SplitbillResult
		if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
			lastErr = err
			e.log.Errorf("gemini attempt=%d unmarshal failed duration=%s cleaned=%q err=%v", attempt, elapsed, receiptprompt.Truncate(cleaned, 500), err)
			continue
		}
		return &out, nil
	}

	e.log.Errorf("gemini extract failed after retries: %v", lastErr)
	return nil, domain.ErrExtractFailed
}

func receiptSchema() *genai.Schema {
	str := &genai.Schema{Type: genai.TypeString}
	nullableTrue := true
	nullableStr := &genai.Schema{Type: genai.TypeString, Nullable: &nullableTrue}
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
	fee := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"type":   str,
			"name":   str,
			"amount": str,
			"rate":   nullableStr,
		},
		Required: []string{"type", "name", "amount"},
	}
	tax := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"amount":         str,
			"service_charge": str,
			"dpp":            nullableStr,
			"name":           str,
			"total_tax":      str,
		},
		Required: []string{"amount", "service_charge", "name", "total_tax"},
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
					"email":        nullableStr,
					"npwp":         nullableStr,
					"phone_number": nullableStr,
					"store_name":   str,
				},
				Required: []string{"address", "store_name"},
			},
			"totals": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"subtotal":       str,
					"discount":       str,
					"fees":           {Type: genai.TypeArray, Items: fee},
					"tax":            tax,
					"service_charge": str,
					"total":          str,
					"payment":        str,
					"change":         nullableStr,
				},
				Required: []string{"subtotal", "discount", "fees", "tax", "service_charge", "total", "payment"},
			},
			"transaction_information": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"date":           str,
					"time":           nullableStr,
					"transaction_id": nullableStr,
				},
				Required: []string{"date"},
			},
			"currency": currency,
			"language": language,
		},
		Required: []string{"items", "store_information", "totals", "transaction_information", "currency", "language"},
	}
}
