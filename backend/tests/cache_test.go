package tests

import (
	"testing"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/cache"
)

func TestCacheSetGet(t *testing.T) {
	c := cache.New()

	c.Set("key1", "value1", 10*time.Minute)

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to be present")
	}
	if val.(string) != "value1" {
		t.Errorf("got %q, want %q", val, "value1")
	}
}

func TestCacheMiss(t *testing.T) {
	c := cache.New()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected cache miss for nonexistent key")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := cache.New()

	c.Set("expiring", "val", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	_, ok := c.Get("expiring")
	if ok {
		t.Fatal("expected expired entry to be a cache miss")
	}
}

func TestCacheDelete(t *testing.T) {
	c := cache.New()
	c.Set("k", "v", time.Minute)
	c.Delete("k")
	_, ok := c.Get("k")
	if ok {
		t.Fatal("expected deleted key to be missing")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := cache.New()
	c.Set("k", "old", time.Minute)
	c.Set("k", "new", time.Minute)
	val, ok := c.Get("k")
	if !ok || val.(string) != "new" {
		t.Errorf("expected 'new', got %v (ok=%v)", val, ok)
	}
}
