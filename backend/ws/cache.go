package ws

import (
	"fmt"
	"sync"
	"time"
)

type Cache struct {
	data            map[string][][]byte
	mu              sync.RWMutex
	done            chan struct{} // For graceful shutdown
	retentionPeriod int64
}

// NewCache creates a new cache instance
func NewCache(retentionPeriod int64) *Cache {
	c := &Cache{
		data:            make(map[string][][]byte),
		done:            make(chan struct{}),
		retentionPeriod: retentionPeriod,
	}

	return c
}

func (c *Cache) Update(key string, value []byte) {
	c.mu.Lock()
	c.data[key] = append(c.data[key], value)
	c.mu.Unlock()
}

func (c *Cache) Get(key string) ([][]byte, bool) {
	c.mu.RLock()
	value, exists := c.data[key]
	c.mu.RUnlock()
	return value, exists
}

func (c *Cache) runCleanup() {
	ticker := time.NewTicker(1 * time.Minute) // Run cleanup every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

func (c *Cache) cleanup() {
	currentEpoch := getCurrentEpoch()
	threshold := currentEpoch - c.retentionPeriod

	// Pre-calculate keys to delete to minimize lock time
	c.mu.RLock()
	keysToDelete := make([]string, 0)
	for key := range c.data {
		var epochKey int64
		if _, err := fmt.Sscanf(key, "%s:updates:%d", &epochKey); err == nil && epochKey < threshold {
			keysToDelete = append(keysToDelete, key)
		}
	}
	c.mu.RUnlock()

	// Only lock for actual deletion
	if len(keysToDelete) > 0 {
		c.mu.Lock()
		for _, key := range keysToDelete {
			delete(c.data, key)
		}
		c.mu.Unlock()
	}
}

func (c *Cache) Close() error {
	close(c.done)
	return nil
}
