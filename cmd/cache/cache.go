package cache

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Body       []byte
	Headers    http.Header
	StatusCode int
	ExpiresAt  time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]Entry
}

func (e Entry) expired() bool {
	return time.Now().After(e.ExpiresAt)
}

func New() *Cache {
	return &Cache{
		items: make(map[string]Entry),
	}
}
func (c *Cache) Get(key string) (Entry, bool) {
	c.mu.RLock()
	entry, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return Entry{}, false
	}

	if entry.expired() {
		c.Delete(key)
		return Entry{}, false
	}

	return entry, true
}

func (c *Cache) Set(key string, entry Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func TTLFromHeaders(h http.Header) time.Duration {
	const defaultTTL = 10 * time.Second

	cc := h.Get("Cache-Control")
	if cc == "" {
		return defaultTTL
	}

	directives := strings.Split(cc, ",")
	for _, d := range directives {
		d = strings.TrimSpace(strings.ToLower(d))

		if d == "no-store" || d == "no-cache" {
			return 0
		}

		if strings.HasPrefix(d, "max-age=") {
			raw := strings.TrimPrefix(d, "max-age=")
			seconds, err := strconv.Atoi(raw)
			if err != nil || seconds < 0 {
				continue
			}
			return time.Duration(seconds) * time.Second
		}
	}

	return defaultTTL
}
