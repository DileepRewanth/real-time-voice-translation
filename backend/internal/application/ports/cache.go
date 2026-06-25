package ports

import (
	"context"
	"time"
)

// Cache defines the outgoing port for caching translation results.
// Implementations can be Redis, in-memory, or any other cache backend.
type Cache interface {
	// Get retrieves a cached value by key. Returns empty string and nil error on cache miss.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value in the cache with the given TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Ping checks if the cache backend is healthy.
	Ping(ctx context.Context) error
}
