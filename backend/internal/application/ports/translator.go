package ports

import (
	"context"

	"github.com/voice-translation/backend/internal/domain"
)

// Translator defines the outgoing port for translation services.
// Any translation backend (Gemini, MyMemory, etc.) must implement this interface.
type Translator interface {
	// Translate performs the translation of the given request.
	Translate(ctx context.Context, req domain.TranslationRequest) (*domain.TranslationResult, error)

	// Name returns the identifier for this translation engine.
	Name() domain.TranslationEngine
}
