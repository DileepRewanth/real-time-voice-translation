package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/voice-translation/backend/internal/application/ports"
	"github.com/voice-translation/backend/internal/config"
	"github.com/voice-translation/backend/internal/domain"
	"github.com/voice-translation/backend/internal/pipeline"
)

// translateRequest is the REST API request body for translation.
type translateRequest struct {
	Text           string   `json:"text"`
	Engine         string   `json:"engine"`
	Context        []string `json:"context,omitempty"`
	TonePreference string   `json:"tone_preference,omitempty"`
}

// translateHandler handles POST /api/v1/translate requests.
func translateHandler(pipe *pipeline.Pipeline, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req translateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{
				Error: "Invalid request body",
				Code:  "INVALID_REQUEST",
				Details: err.Error(),
			})
			return
		}

		// Validate
		if req.Text == "" {
			writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{
				Error: "Text field is required",
				Code:  "MISSING_FIELD",
			})
			return
		}

		if len(req.Text) > 5000 {
			writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{
				Error: "Text exceeds maximum length of 5000 characters",
				Code:  "TEXT_TOO_LONG",
			})
			return
		}

		// Determine engine
		engine := domain.EngineMyMemory
		if req.Engine == "gemini" {
			engine = domain.EngineGemini
		}

		// Build domain request
		domainReq := domain.TranslationRequest{
			Text:           req.Text,
			Engine:         engine,
			Context:        req.Context,
			TonePreference: req.TonePreference,
		}

		// Execute pipeline
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		result, err := pipe.Process(ctx, domainReq)
		if err != nil {
			logger.Error("translation pipeline failed",
				"error", err,
				"text_length", len(req.Text),
				"engine", engine,
			)
			writeJSON(w, http.StatusInternalServerError, domain.ErrorResponse{
				Error: "Translation failed",
				Code:  "TRANSLATION_ERROR",
				Details: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// healthHandler returns the health status of the service.
func healthHandler(cache ports.Cache, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := make(map[string]string)

		// Check cache health
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := cache.Ping(ctx); err != nil {
			services["cache"] = "unhealthy: " + err.Error()
		} else {
			services["cache"] = "healthy"
		}

		services["gemini"] = "not_configured"
		if cfg.IsGeminiConfigured() {
			services["gemini"] = "configured"
		}

		status := domain.HealthStatus{
			Status:    "ok",
			Timestamp: time.Now(),
			Services:  services,
			Version:   cfg.Version,
		}

		writeJSON(w, http.StatusOK, status)
	}
}

// readinessHandler checks if the service is ready to accept traffic.
func readinessHandler(cache ports.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// For readiness, we just need the server to be up
		// Cache is optional (fallback exists)
		_ = cache.Ping(ctx)

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
	}
}

// configHandler returns the available configuration.
func configHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		engines := []domain.TranslationEngine{domain.EngineMyMemory}
		if cfg.IsGeminiConfigured() {
			engines = append(engines, domain.EngineGemini)
		}

		resp := domain.ConfigResponse{
			AvailableEngines: engines,
			DefaultEngine:    domain.TranslationEngine(cfg.DefaultEngine),
			GeminiConfigured: cfg.IsGeminiConfigured(),
			WebSocketPath:    "/api/v1/ws",
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
