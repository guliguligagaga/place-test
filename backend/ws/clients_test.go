package ws

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestGenerateClientID(t *testing.T) {
	atomic.StoreUint32(&clientCounter, 0)

	id1 := generateClientID()
	id2 := generateClientID()
	id3 := generateClientID()

	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Errorf("Generated IDs are not unique: %032b, %032b, %032b", id1, id2, id3)
	}

}

func TestGenerateClientIDConcurrency(t *testing.T) {
	atomic.StoreUint32(&clientCounter, 0)

	numGoroutines := 1000
	numIDsPerGoroutine := 100

	var wg sync.WaitGroup
	ids := make([]uint64, numGoroutines*numIDsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for j := 0; j < numIDsPerGoroutine; j++ {
				ids[offset+j] = generateClientID()
			}
		}(i * numIDsPerGoroutine)
	}

	wg.Wait()

	idMap := make(map[uint64]bool)
	for _, id := range ids {
		if idMap[id] {
			t.Errorf("Duplicate ID found: %032b", id)
		}
		idMap[id] = true
	}
}

func BenchmarkGenerateClientID(b *testing.B) {
	atomic.StoreUint32(&clientCounter, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		generateClientID()
	}
}
