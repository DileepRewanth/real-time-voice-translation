package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache implements the Cache port using an in-memory map.
// Used as a fallback when Redis is unavailable.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewMemoryCache creates an in-memory cache with a max entry limit.
func NewMemoryCache(maxSize int) *MemoryCache {
	mc := &MemoryCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
	// Start background cleanup goroutine
	go mc.cleanup()
	return mc
}

// Get retrieves a value from the in-memory cache. Returns empty string on miss or expiry.
func (m *MemoryCache) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return "", nil
	}
	if time.Now().After(entry.expiresAt) {
		return "", nil // expired
	}
	return entry.value, nil
}

// Set stores a value in the in-memory cache with the given TTL.
// If the cache is full, the oldest entry is evicted.
func (m *MemoryCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Evict if at capacity (simple eviction: remove first expired, then oldest)
	if len(m.entries) >= m.maxSize {
		m.evictOne()
	}

	m.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Ping always returns nil for in-memory cache (it's always "healthy").
func (m *MemoryCache) Ping(_ context.Context) error {
	return nil
}

// evictOne removes one entry — preferring expired entries, then the oldest.
func (m *MemoryCache) evictOne() {
	now := time.Now()
	var oldestKey string
	var oldestTime time.Time
	first := true

	for k, v := range m.entries {
		// Remove expired entry immediately
		if now.After(v.expiresAt) {
			delete(m.entries, k)
			return
		}
		// Track oldest
		if first || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
			first = false
		}
	}
	if oldestKey != "" {
		delete(m.entries, oldestKey)
	}
}

// cleanup periodically removes expired entries.
func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for k, v := range m.entries {
			if now.After(v.expiresAt) {
				delete(m.entries, k)
			}
		}
		m.mu.Unlock()
	}
}
