package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voice-translation/backend/internal/adapters/cache"
	adapthttp "github.com/voice-translation/backend/internal/adapters/http"
	"github.com/voice-translation/backend/internal/adapters/translator"
	"github.com/voice-translation/backend/internal/application/ports"
	"github.com/voice-translation/backend/internal/application/usecase"
	"github.com/voice-translation/backend/internal/config"
	"github.com/voice-translation/backend/internal/domain"
	"github.com/voice-translation/backend/internal/pipeline"
)

func main() {
	// --- Load Configuration ---
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// --- Setup Structured Logger ---
	var logHandler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}
	if cfg.LogFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting voice translation server",
		"version", cfg.Version,
		"port", cfg.ServerPort,
		"log_level", cfg.LogLevel,
	)

	// --- Initialize Cache (Redis with in-memory fallback) ---
	var cacheAdapter ports.Cache
	redisCache, err := cache.NewRedisCache(cfg.RedisURL, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Warn("redis connection failed, using in-memory cache",
			"error", err,
			"redis_url", cfg.RedisURL,
		)
		cacheAdapter = cache.NewMemoryCache(10000) // 10k max entries
	} else {
		logger.Info("redis connected successfully", "url", cfg.RedisURL)
		cacheAdapter = redisCache
	}

	// --- Initialize Translators ---
	translators := make(map[domain.TranslationEngine]ports.Translator)

	// MyMemory is always available (no API key required)
	translators[domain.EngineMyMemory] = translator.NewMyMemoryTranslator(logger)
	logger.Info("mymemory translator initialized")

	// Gemini requires an API key
	if cfg.IsGeminiConfigured() {
		translators[domain.EngineGemini] = translator.NewGeminiTranslator(
			cfg.GeminiAPIKey, cfg.GeminiModel, logger,
		)
		logger.Info("gemini translator initialized", "model", cfg.GeminiModel)
	} else {
		logger.Warn("gemini API key not configured — gemini engine unavailable")
	}

	// --- Build Use Case ---
	translateUC := usecase.NewTranslateUseCase(translators, cacheAdapter, cfg.CacheTTL, logger)

	// --- Build Pipeline ---
	pipe := pipeline.NewPipeline(translateUC, logger)

	// --- Initialize WebSocket Hub ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := adapthttp.NewHub(logger)
	adapthttp.SetHub(hub)
	go hub.Run(ctx)

	// --- Build HTTP Router ---
	router := adapthttp.NewRouter(cfg, pipe, cacheAdapter, logger)

	// --- Start HTTP Server ---
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for WebSocket upgrades
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutdown signal received", "signal", sig.String())

	// Give in-flight requests up to 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel WebSocket hub context
	cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Close Redis if applicable
	if rc, ok := cacheAdapter.(*cache.RedisCache); ok {
		if err := rc.Close(); err != nil {
			logger.Error("redis close error", "error", err)
		}
	}

	logger.Info("server stopped gracefully")
}

// parseLogLevel converts a string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
