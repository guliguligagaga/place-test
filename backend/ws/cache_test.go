package ws

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestCacheCleanup(t *testing.T) {
	c := NewCache(2) // 2 epoch retention
	currentEpoch := getCurrentEpoch()

	// Setup test data
	validKey := fmt.Sprintf("%s:updates:%d", gridKey, currentEpoch-1)
	expiredKey := fmt.Sprintf("%s:updates:%d", gridKey, currentEpoch-3)

	c.Update(validKey, []byte("valid"))
	c.Update(expiredKey, []byte("expired"))

	c.cleanup()

	t.Run("Retains recent entries", func(t *testing.T) {
		_, exists := c.Get(validKey)
		assert.True(t, exists)
	})

	t.Run("Removes expired entries", func(t *testing.T) {
		_, exists := c.Get(expiredKey)
		assert.False(t, exists)
	})
}

func TestCacheKeyParsing(t *testing.T) {
	key := "grid123:updates:1715616000000"
	expectedEpoch := int64(1715616000000)

	epochStart := strings.LastIndex(key, ":")
	epochStr := key[epochStart+1:]
	epoch, err := strconv.ParseInt(epochStr, 10, 64)

	assert.Nil(t, err)
	assert.Equal(t, expectedEpoch, epoch)
}
