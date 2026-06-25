package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"github.com/voice-translation/backend/internal/domain"
)

// GeminiTranslator implements the Translator port using the Google Gemini API.
type GeminiTranslator struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewGeminiTranslator creates a Gemini-backed translator.
func NewGeminiTranslator(apiKey, model string, logger *slog.Logger) *GeminiTranslator {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// If no API key is provided but GOOGLE_APPLICATION_CREDENTIALS is set, use OAuth2 client
	if apiKey == "" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		oauthClient, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/generative-language")
		if err == nil {
			client = oauthClient
		} else {
			logger.Error("failed to create oauth client", "error", err)
		}
	}

	return &GeminiTranslator{
		apiKey:     apiKey,
		model:      model,
		httpClient: client,
		logger:     logger,
	}
}

// Name returns the engine identifier.
func (g *GeminiTranslator) Name() domain.TranslationEngine {
	return domain.EngineGemini
}

// geminiRequest represents the Gemini API request body.
type geminiRequest struct {
	Contents         []geminiContent       `json:"contents"`
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationCfg `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationCfg struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// geminiResponse represents the Gemini API response.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// Translate sends text to the Gemini API for context-aware English→Hindi translation.
func (g *GeminiTranslator) Translate(ctx context.Context, req domain.TranslationRequest) (*domain.TranslationResult, error) {
	start := time.Now()

	systemPrompt := g.buildSystemPrompt(req.TonePreference)
	userPrompt := g.buildUserPrompt(req.Text, req.Context)

	apiReq := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: userPrompt}},
				Role:  "user",
			},
		},
		GenerationConfig: &geminiGenerationCfg{
			Temperature:     0.3,
			MaxOutputTokens: 1024,
		},
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", g.model)
	if g.apiKey != "" {
		url = fmt.Sprintf("%s?key=%s", url, g.apiKey)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		g.logger.Error("gemini API error",
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned empty response")
	}

	translatedText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	latency := time.Since(start).Milliseconds()

	g.logger.Info("gemini translation completed",
		"input_length", len(req.Text),
		"output_length", len(translatedText),
		"latency_ms", latency,
	)

	return &domain.TranslationResult{
		OriginalText:   req.Text,
		ProcessedText:  req.Text,
		TranslatedText: translatedText,
		Engine:         domain.EngineGemini,
		Cached:         false,
		Latency: domain.LatencyBreakdown{
			TranslateMs: latency,
		},
		Timestamp: time.Now(),
	}, nil
}

// buildSystemPrompt creates the Gemini system instruction for translation.
func (g *GeminiTranslator) buildSystemPrompt(tone string) string {
	toneInstruction := "Use a natural, conversational Hindi tone."
	if tone == "formal" {
		toneInstruction = "Use formal Hindi (आप form). Suitable for professional/business contexts."
	} else if tone == "casual" {
		toneInstruction = "Use casual/informal Hindi (तुम/तू form). Suitable for friendly conversation."
	}

	return fmt.Sprintf(`You are an expert English-to-Hindi translator for a real-time voice translation system. Follow these rules strictly:

1. TRANSLATION: Translate the given English text into natural, fluent Hindi. Output ONLY the Hindi translation — no explanations, notes, or English text.

2. NAMED ENTITIES: Preserve proper nouns and brand names as-is. Do NOT translate or transliterate names like "Google Meet", "Zoom", "WhatsApp", "Asterisk", "Kubernetes", "Docker", etc. Keep them in English within the Hindi sentence.

3. NUMBERS & DATES: Localize correctly:
   - "5 PM" / "5:00 p.m." → "शाम 5 बजे" (IMPORTANT: Do NOT use "5:00 बजे". The trailing ":00" causes TTS engines to stutter. Strip the zeros).
   - "10 AM" → "सुबह 10 बजे"  
   - "March 15" → "15 मार्च"
   - "ETA" → "अनुमानित समय"
   - "Q3" → "तीसरी तिमाही"
   - Currency and percentages should use Indian conventions.

4. FILLERS: If the input contains filler words like "uh", "um", "you know", "like", "basically", "I mean" — ignore them entirely. Do not translate fillers.

5. TONE: %s

6. INCOMPLETE SENTENCES: If the input seems like a fragment or incomplete thought, still translate what is given as naturally as possible.

7. ABBREVIATIONS: Common abbreviations should be expanded in Hindi:
   - "ASAP" → "जल्द से जल्द"
   - "FYI" → "आपकी जानकारी के लिए"
   - "BTW" → "वैसे"
   
8. MEETING PHRASES: Accurately translate common video call/meeting phrases into natural Hindi:
   - "Am I audible?" / "Can you hear me?" → "क्या मेरी आवाज़ आ रही है?"
   - "Can you see my screen?" → "क्या आपको मेरी स्क्रीन दिख रही है?"
   - "You are on mute" → "आप म्यूट पर हैं"
   - "Let's wait for others" → "बाकी लोगों का इंतज़ार करते हैं"`, toneInstruction)
}

// buildUserPrompt constructs the translation prompt with optional context.
func (g *GeminiTranslator) buildUserPrompt(text string, contextHistory []string) string {
	if len(contextHistory) == 0 {
		return fmt.Sprintf("Translate to Hindi:\n%s", text)
	}

	contextStr := strings.Join(contextHistory, "\n")
	return fmt.Sprintf("Previous context (for reference only, do not re-translate):\n%s\n\nTranslate this new sentence to Hindi:\n%s", contextStr, text)
}
