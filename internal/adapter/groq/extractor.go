package groq

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/receiptprompt"
	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/sirupsen/logrus"
)

const (
	defaultBaseURL = "https://api.groq.com/openai/v1"
	maxAttempts    = 2
)

type Extractor struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
	client  *http.Client
	log     *logrus.Logger
}

func NewExtractor(apiKey, model string, timeout time.Duration, log *logrus.Logger) (*Extractor, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GROQ_API_KEY is required")
	}
	if model == "" {
		model = "llama-3.1-8b-instant"
	}
	if timeout < time.Second {
		timeout = 25 * time.Second
	}
	return &Extractor{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
		timeout: timeout,
		client:  &http.Client{},
		log:     log,
	}, nil
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(image))
	payload := chatRequest{
		Model: e.model,
		Messages: []chatMessage{{
			Role: "user",
			Content: []contentPart{
				{Type: "text", Text: receiptprompt.Extract},
				{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
			},
		}},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    0.1,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		e.log.Errorf("groq marshal request failed: %v", err)
		return nil, domain.ErrExtractFailed
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		raw, err := e.doChat(ctx, body)
		if err != nil {
			e.log.Errorf("groq attempt=%d generate failed: %v", attempt, err)
			lastErr = err
			continue
		}

		cleaned := receiptprompt.CleanJSON(raw)
		e.log.Debugf("groq attempt=%d raw_len=%d cleaned_len=%d", attempt, len(raw), len(cleaned))
		if cleaned == "" {
			lastErr = fmt.Errorf("empty response")
			e.log.Warnf("groq attempt=%d empty cleaned response", attempt)
			continue
		}

		var out domain.SplitbillResult
		if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
			lastErr = err
			e.log.Errorf("groq attempt=%d unmarshal failed cleaned=%q err=%v", attempt, receiptprompt.Truncate(cleaned, 500), err)
			continue
		}
		return &out, nil
	}

	e.log.Errorf("groq extract failed after retries: %v", lastErr)
	return nil, domain.ErrExtractFailed
}

func (e *Extractor) doChat(ctx context.Context, body []byte) (string, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, e.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, receiptprompt.Truncate(string(respBody), 300))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
