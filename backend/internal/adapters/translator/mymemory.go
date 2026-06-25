package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/voice-translation/backend/internal/domain"
)

// MyMemoryTranslator implements the Translator port using the MyMemory free API.
type MyMemoryTranslator struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewMyMemoryTranslator creates a MyMemory-backed translator.
func NewMyMemoryTranslator(logger *slog.Logger) *MyMemoryTranslator {
	return &MyMemoryTranslator{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		logger: logger,
	}
}

// Name returns the engine identifier.
func (m *MyMemoryTranslator) Name() domain.TranslationEngine {
	return domain.EngineMyMemory
}

// myMemoryResponse represents the MyMemory API response.
type myMemoryResponse struct {
	ResponseData struct {
		TranslatedText string  `json:"translatedText"`
		Match          float64 `json:"match"`
	} `json:"responseData"`
	ResponseStatus int    `json:"responseStatus"`
	ResponseDetails string `json:"responseDetails"`
}

// Translate sends text to the MyMemory API for English→Hindi translation.
func (m *MyMemoryTranslator) Translate(ctx context.Context, req domain.TranslationRequest) (*domain.TranslationResult, error) {
	start := time.Now()

	apiURL := fmt.Sprintf(
		"https://api.mymemory.translated.net/get?q=%s&langpair=en|hi&mt=1",
		url.QueryEscape(req.Text),
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mymemory API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("mymemory API error",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("mymemory API returned status %d", resp.StatusCode)
	}

	var mmResp myMemoryResponse
	if err := json.Unmarshal(body, &mmResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if mmResp.ResponseStatus != 200 {
		return nil, fmt.Errorf("mymemory error: %s (status %d)", mmResp.ResponseDetails, mmResp.ResponseStatus)
	}

	translatedText := mmResp.ResponseData.TranslatedText
	latency := time.Since(start).Milliseconds()

	m.logger.Info("mymemory translation completed",
		"input_length", len(req.Text),
		"output_length", len(translatedText),
		"latency_ms", latency,
		"match", mmResp.ResponseData.Match,
	)

	return &domain.TranslationResult{
		OriginalText:   req.Text,
		ProcessedText:  req.Text,
		TranslatedText: translatedText,
		Engine:         domain.EngineMyMemory,
		Cached:         false,
		Latency: domain.LatencyBreakdown{
			TranslateMs: latency,
		},
		Timestamp: time.Now(),
	}, nil
}
