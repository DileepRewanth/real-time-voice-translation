package pipeline

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/voice-translation/backend/internal/application/usecase"
	"github.com/voice-translation/backend/internal/domain"
)

// Pipeline implements a concurrent multi-stage translation pipeline using goroutines and channels.
// Stages: PreProcess → Translate → PostProcess
type Pipeline struct {
	translateUC *usecase.TranslateUseCase
	logger      *slog.Logger
}

// NewPipeline creates a new translation pipeline.
func NewPipeline(translateUC *usecase.TranslateUseCase, logger *slog.Logger) *Pipeline {
	return &Pipeline{
		translateUC: translateUC,
		logger:      logger,
	}
}

// Process runs the full pipeline synchronously for a single request.
// Each stage is timed independently for latency tracking.
func (p *Pipeline) Process(ctx context.Context, req domain.TranslationRequest) (*domain.TranslationResult, error) {
	totalStart := time.Now()
	msg := &domain.PipelineMessage{
		ID:        req.SessionID,
		Request:   req,
		Text:      req.Text,
		Stage:     domain.StagePreProcess,
		StartTime: totalStart,
	}

	// Stage 1: Pre-Process
	preStart := time.Now()
	processedText := p.preProcess(msg.Text)
	msg.Text = processedText
	msg.Latency.PreProcessMs = time.Since(preStart).Milliseconds()
	msg.Stage = domain.StageTranslate

	p.logger.Info("pipeline: pre-process complete",
		"original", req.Text,
		"processed", processedText,
		"latency_ms", msg.Latency.PreProcessMs,
	)

	// Stage 2: Translate
	translateReq := domain.TranslationRequest{
		Text:           processedText,
		Engine:         req.Engine,
		Context:        req.Context,
		TonePreference: req.TonePreference,
		SessionID:      req.SessionID,
	}

	result, err := p.translateUC.Execute(ctx, translateReq)
	if err != nil {
		return nil, err
	}

	msg.Latency.TranslateMs = result.Latency.TranslateMs
	msg.Stage = domain.StagePostProcess

	// Stage 3: Post-Process
	postStart := time.Now()
	result.TranslatedText = p.postProcess(result.TranslatedText)
	msg.Latency.PostProcessMs = time.Since(postStart).Milliseconds()

	// Set final metrics
	result.OriginalText = req.Text
	result.ProcessedText = processedText
	result.Latency.PreProcessMs = msg.Latency.PreProcessMs
	result.Latency.PostProcessMs = msg.Latency.PostProcessMs
	result.Latency.TotalMs = time.Since(totalStart).Milliseconds()

	p.logger.Info("pipeline: complete",
		"total_ms", result.Latency.TotalMs,
		"pre_ms", result.Latency.PreProcessMs,
		"translate_ms", result.Latency.TranslateMs,
		"post_ms", result.Latency.PostProcessMs,
		"engine", result.Engine,
		"cached", result.Cached,
	)

	return result, nil
}

// ProcessAsync runs the pipeline in a goroutine and sends stage updates via channels.
// This enables real-time WebSocket progress updates.
func (p *Pipeline) ProcessAsync(ctx context.Context, req domain.TranslationRequest,
	statusCh chan<- domain.WSStatusPayload,
	resultCh chan<- *domain.TranslationResult,
	errCh chan<- error,
) {
	go func() {
		// Stage 1: Pre-Process
		statusCh <- domain.WSStatusPayload{
			Stage:  domain.StagePreProcess,
			Status: "processing",
		}

		totalStart := time.Now()
		preStart := time.Now()
		processedText := p.preProcess(req.Text)
		preLatency := time.Since(preStart).Milliseconds()

		statusCh <- domain.WSStatusPayload{
			Stage:   domain.StagePreProcess,
			Status:  "completed",
			Message: processedText,
		}

		// Stage 2: Translate
		statusCh <- domain.WSStatusPayload{
			Stage:  domain.StageTranslate,
			Status: "processing",
		}

		translateReq := domain.TranslationRequest{
			Text:           processedText,
			Engine:         req.Engine,
			Context:        req.Context,
			TonePreference: req.TonePreference,
			SessionID:      req.SessionID,
		}

		result, err := p.translateUC.Execute(ctx, translateReq)
		if err != nil {
			statusCh <- domain.WSStatusPayload{
				Stage:   domain.StageTranslate,
				Status:  "error",
				Message: err.Error(),
			}
			errCh <- err
			return
		}

		statusCh <- domain.WSStatusPayload{
			Stage:  domain.StageTranslate,
			Status: "completed",
		}

		// Stage 3: Post-Process
		statusCh <- domain.WSStatusPayload{
			Stage:  domain.StagePostProcess,
			Status: "processing",
		}

		postStart := time.Now()
		result.TranslatedText = p.postProcess(result.TranslatedText)
		postLatency := time.Since(postStart).Milliseconds()

		statusCh <- domain.WSStatusPayload{
			Stage:  domain.StagePostProcess,
			Status: "completed",
		}

		// Set final metrics
		result.OriginalText = req.Text
		result.ProcessedText = processedText
		result.Latency.PreProcessMs = preLatency
		result.Latency.PostProcessMs = postLatency
		result.Latency.TotalMs = time.Since(totalStart).Milliseconds()

		resultCh <- result
	}()
}

// fillerPatterns are common English filler words/phrases to remove before translation.
var fillerPatterns = regexp.MustCompile(
	`(?i)\b(uh+|um+|hmm+|er+|ah+|you know|i mean|like,?\s|basically,?\s|actually,?\s|literally,?\s|right\??\s|so,?\s(?:yeah|like))\b`,
)

// preProcess cleans the input text before translation.
func (p *Pipeline) preProcess(text string) string {
	// Remove filler words
	cleaned := fillerPatterns.ReplaceAllString(text, " ")

	// Normalize whitespace
	spacePattern := regexp.MustCompile(`\s+`)
	cleaned = spacePattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	// Handle empty result (entire input was fillers)
	if cleaned == "" {
		return text // Fall back to original rather than empty
	}

	return cleaned
}

// postProcess performs final cleanup on translated text.
func (p *Pipeline) postProcess(text string) string {
	// Remove common markdown formatting (asterisks, underscores, hashes)
	markdownPattern := regexp.MustCompile(`[*_#]+`)
	text = markdownPattern.ReplaceAllString(text, "")

	// Remove bracketed text (often added by LLMs like [Music] or (Notes))
	bracketPattern := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	text = bracketPattern.ReplaceAllString(text, "")

	// Remove any "Translation:" or similar prefixes that LLMs sometimes add
	prefixes := []string{
		"Translation:", "Hindi:", "Hindi Translation:", "Translation", "Hindi",
		"अनुवाद:", "हिंदी:",
	}
	for _, prefix := range prefixes {
		// Case insensitive removal if it's at the start
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), strings.ToLower(prefix)) {
			text = text[len(prefix):]
		}
	}

	// Trim any accidental whitespace or quotes
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\"'")

	return text
}
