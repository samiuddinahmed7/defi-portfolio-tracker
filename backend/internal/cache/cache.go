// Package cache provides a lightweight in-memory cache suitable for this
// personal project. Redis would add operational complexity without meaningful
// benefit at this scale. The tradeoff: cache state is lost on restart and
// is not shared across processes, but for a single-instance personal tool
// that is entirely acceptable.
package cache

import (
	"sync"
	"time"
)

// Entry stores a cached value together with its expiry.
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is a simple thread-safe in-memory key-value store with TTL support.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// New returns an initialised Cache. Call StartEviction to periodically purge
// expired entries (optional; Get already skips expired values).
func New() *Cache {
	return &Cache{entries: make(map[string]Entry)}
}

// Set stores value under key for the given TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value. ok is false when the key is absent or expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, found := c.entries[key]
	if !found {
		return nil, false
	}
	if time.Now().After(e.ExpiresAt) {
		return nil, false
	}
	return e.Value, true
}

// Delete removes a key unconditionally.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// StartEviction launches a background goroutine that removes expired entries
// on the given interval. It stops when stopCh is closed.
func (c *Cache) StartEviction(interval time.Duration, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.evict()
			case <-stopCh:
				return
			}
		}
	}()
}

func (c *Cache) evict() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if now.After(e.ExpiresAt) {
			delete(c.entries, k)
		}
	}
}
