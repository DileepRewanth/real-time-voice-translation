package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/voice-translation/backend/internal/config"
	"github.com/voice-translation/backend/internal/pipeline"
	"github.com/voice-translation/backend/internal/application/ports"
)

// NewRouter creates and configures the chi router with all middleware and routes.
func NewRouter(
	cfg *config.Config,
	pipe *pipeline.Pipeline,
	cache ports.Cache,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Middleware chain
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))

	// Custom structured logging middleware
	r.Use(structuredLogger(logger))

	// CORS middleware
	r.Use(corsMiddleware(cfg.AllowedOrigins))

	// Health endpoints (outside versioned API — standard practice)
	r.Get("/health", healthHandler(cache, cfg))
	r.Get("/health/ready", readinessHandler(cache))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/translate", translateHandler(pipe, logger))
		r.Get("/config", configHandler(cfg))
		r.Get("/ws", wsHandler(pipe, logger))
	})

	return r
}

// structuredLogger is a middleware that logs each request using slog.
func structuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("request completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
					"remote_addr", r.RemoteAddr,
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// corsMiddleware handles CORS headers for cross-origin requests.
func corsMiddleware(allowedOrigins []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range allowedOrigins {
				if strings.TrimSpace(o) == origin || strings.TrimSpace(o) == "*" {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
