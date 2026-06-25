package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/voice-translation/backend/internal/application/ports"
	"github.com/voice-translation/backend/internal/domain"
)

// TranslateUseCase orchestrates the translation workflow.
// It handles caching, engine routing, and circuit breaker logic.
type TranslateUseCase struct {
	translators map[domain.TranslationEngine]ports.Translator
	cache       ports.Cache
	cacheTTL    time.Duration
	logger      *slog.Logger

	// Circuit breaker state
	mu             sync.RWMutex
	failureCounts  map[domain.TranslationEngine]int
	maxFailures    int
	circuitResetAt map[domain.TranslationEngine]time.Time
}

// NewTranslateUseCase creates the translation use case with injected dependencies.
func NewTranslateUseCase(
	translators map[domain.TranslationEngine]ports.Translator,
	cache ports.Cache,
	cacheTTL time.Duration,
	logger *slog.Logger,
) *TranslateUseCase {
	return &TranslateUseCase{
		translators:    translators,
		cache:          cache,
		cacheTTL:       cacheTTL,
		logger:         logger,
		failureCounts:  make(map[domain.TranslationEngine]int),
		maxFailures:    3,
		circuitResetAt: make(map[domain.TranslationEngine]time.Time),
	}
}

// Execute runs the translation workflow: cache check → translate → cache store.
func (uc *TranslateUseCase) Execute(ctx context.Context, req domain.TranslationRequest) (*domain.TranslationResult, error) {
	engine := req.Engine
	if engine == "" {
		engine = domain.EngineMyMemory
	}

	// Check circuit breaker — fall back if engine is tripped
	if uc.isCircuitOpen(engine) {
		uc.logger.Warn("circuit breaker open, falling back",
			"tripped_engine", engine,
			"fallback_engine", domain.EngineMyMemory,
		)
		engine = uc.getFallbackEngine(engine)
	}

	// Check cache
	cacheKey := uc.buildCacheKey(req.Text, engine)
	if cached, err := uc.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		uc.logger.Info("cache hit",
			"engine", engine,
			"text_length", len(req.Text),
		)
		return &domain.TranslationResult{
			OriginalText:   req.Text,
			ProcessedText:  req.Text,
			TranslatedText: cached,
			Engine:         engine,
			Cached:         true,
			Latency:        domain.LatencyBreakdown{TranslateMs: 0},
			Timestamp:      time.Now(),
		}, nil
	}

	// Get translator
	translator, ok := uc.translators[engine]
	if !ok {
		return nil, fmt.Errorf("unknown translation engine: %s", engine)
	}

	// Execute translation
	start := time.Now()
	result, err := translator.Translate(ctx, req)
	if err != nil {
		uc.recordFailure(engine)
		uc.logger.Error("translation failed",
			"engine", engine,
			"error", err,
		)

		// Try fallback engine
		fallback := uc.getFallbackEngine(engine)
		if fallback != engine {
			uc.logger.Info("attempting fallback translation", "fallback_engine", fallback)
			if fbTranslator, ok := uc.translators[fallback]; ok {
				req.Engine = fallback
				result, err = fbTranslator.Translate(ctx, req)
				if err != nil {
					return nil, fmt.Errorf("fallback translation also failed: %w", err)
				}
			}
		} else {
			return nil, err
		}
	} else {
		uc.recordSuccess(engine)
	}

	result.Latency.TranslateMs = time.Since(start).Milliseconds()

	// Store in cache
	if err := uc.cache.Set(ctx, cacheKey, result.TranslatedText, uc.cacheTTL); err != nil {
		uc.logger.Warn("failed to cache translation", "error", err)
		// Non-fatal: continue even if caching fails
	}

	return result, nil
}

// buildCacheKey creates a deterministic cache key from the input text and engine.
func (uc *TranslateUseCase) buildCacheKey(text string, engine domain.TranslationEngine) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("translation:%x:%s", hash[:8], engine)
}

// isCircuitOpen checks if the circuit breaker is open for an engine.
func (uc *TranslateUseCase) isCircuitOpen(engine domain.TranslationEngine) bool {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	count, ok := uc.failureCounts[engine]
	if !ok || count < uc.maxFailures {
		return false
	}

	// Check if reset time has passed
	if resetAt, ok := uc.circuitResetAt[engine]; ok {
		if time.Now().After(resetAt) {
			return false // Allow a retry (half-open state)
		}
	}

	return true
}

// recordFailure increments the failure count for an engine.
func (uc *TranslateUseCase) recordFailure(engine domain.TranslationEngine) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.failureCounts[engine]++
	if uc.failureCounts[engine] >= uc.maxFailures {
		// Set circuit reset time (30 seconds from now)
		uc.circuitResetAt[engine] = time.Now().Add(30 * time.Second)
		uc.logger.Warn("circuit breaker tripped",
			"engine", engine,
			"failures", uc.failureCounts[engine],
			"reset_at", uc.circuitResetAt[engine],
		)
	}
}

// recordSuccess resets the failure count on a successful translation.
func (uc *TranslateUseCase) recordSuccess(engine domain.TranslationEngine) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.failureCounts[engine] = 0
	delete(uc.circuitResetAt, engine)
}

// getFallbackEngine returns the fallback engine for the given engine.
func (uc *TranslateUseCase) getFallbackEngine(engine domain.TranslationEngine) domain.TranslationEngine {
	if engine == domain.EngineGemini {
		return domain.EngineMyMemory
	}
	return domain.EngineGemini
}
