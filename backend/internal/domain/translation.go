package domain

import "time"

// TranslationEngine represents the available translation backends.
type TranslationEngine string

const (
	EngineGemini   TranslationEngine = "gemini"
	EngineMyMemory TranslationEngine = "mymemory"
)

// PipelineStage represents a stage in the translation pipeline.
type PipelineStage string

const (
	StagePreProcess  PipelineStage = "pre_process"
	StageTranslate   PipelineStage = "translate"
	StagePostProcess PipelineStage = "post_process"
)

// TranslationRequest represents an incoming translation request.
type TranslationRequest struct {
	Text           string            `json:"text"`
	Engine         TranslationEngine `json:"engine"`
	Context        []string          `json:"context,omitempty"`
	TonePreference string            `json:"tone_preference,omitempty"` // "formal" or "casual"
	SessionID      string            `json:"session_id,omitempty"`
}

// TranslationResult represents the output of a translation.
type TranslationResult struct {
	OriginalText   string            `json:"original_text"`
	ProcessedText  string            `json:"processed_text"`
	TranslatedText string            `json:"translated_text"`
	Engine         TranslationEngine `json:"engine"`
	Cached         bool              `json:"cached"`
	Latency        LatencyBreakdown  `json:"latency"`
	Timestamp      time.Time         `json:"timestamp"`
}

// LatencyBreakdown tracks timing for each pipeline stage.
type LatencyBreakdown struct {
	PreProcessMs  int64 `json:"pre_process_ms"`
	TranslateMs   int64 `json:"translate_ms"`
	PostProcessMs int64 `json:"post_process_ms"`
	TotalMs       int64 `json:"total_ms"`
}

// PipelineMessage is the data envelope flowing through pipeline channels.
type PipelineMessage struct {
	ID        string
	Request   TranslationRequest
	Text      string
	Stage     PipelineStage
	StartTime time.Time
	Latency   LatencyBreakdown
	Error     error
}

// WSMessage represents a WebSocket message between client and server.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSTranslatePayload is the payload for a "translate" WebSocket message.
type WSTranslatePayload struct {
	Text           string   `json:"text"`
	Engine         string   `json:"engine,omitempty"`
	Context        []string `json:"context,omitempty"`
	TonePreference string   `json:"tone_preference,omitempty"`
}

// WSStatusPayload is sent to clients to report pipeline stage progress.
type WSStatusPayload struct {
	Stage   PipelineStage `json:"stage"`
	Status  string        `json:"status"` // "processing", "completed", "error"
	Message string        `json:"message,omitempty"`
}

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

// ConfigResponse is returned by the config endpoint.
type ConfigResponse struct {
	AvailableEngines []TranslationEngine `json:"available_engines"`
	DefaultEngine    TranslationEngine   `json:"default_engine"`
	GeminiConfigured bool                `json:"gemini_configured"`
	WebSocketPath    string              `json:"websocket_path"`
}

// ErrorResponse is a standardized API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}
