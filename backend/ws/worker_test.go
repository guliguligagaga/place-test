package ws

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool()
	assert.NotNil(t, pool)
	assert.Equal(t, numWorkers, pool.numWorkers)
	assert.Len(t, pool.workers, numWorkers)

	for i, worker := range pool.workers {
		assert.NotNil(t, worker)
		assert.Equal(t, i, worker.id)
		assert.NotNil(t, worker.clients)
		assert.NotNil(t, worker.messages)
		assert.NotNil(t, worker.done)
		assert.NotNil(t, worker.metrics)
	}
}

func TestWorkerClientManagement(t *testing.T) {
	worker := &Worker{
		id:      0,
		clients: make(map[uint64]*Client),
		metrics: &WorkerMetrics{},
	}

	now := time.Now().Unix()

	clientID := generateClientID()
	client := &Client{
		ID:   clientID,
		send: make(chan []byte, clientQueueSize),
		done: make(chan struct{}),
	}
	client.lastPing.Store(now)
	worker.clients[clientID] = client

	assert.Contains(t, worker.clients, clientID)
	assert.Equal(t, client, worker.clients[clientID])

	client.lastPing.Store(now - int64(readTimeout.Seconds()) - 10)

	worker.cleanupInactiveClients()

	assert.NotContains(t, worker.clients, clientID)
}

func TestWorkerQueueSize(t *testing.T) {
	worker := &Worker{
		id:       0,
		messages: make(chan []byte, workerQueueSize),
		metrics:  &WorkerMetrics{},
	}

	// Test that queue accepts messages up to size
	for i := 0; i < workerQueueSize; i++ {
		select {
		case worker.messages <- []byte("test"):
			// Message accepted
		default:
			t.Fatal("Queue should accept message")
		}
	}

	// Verify queue is now full
	select {
	case worker.messages <- []byte("test"):
		t.Fatal("Queue should be full")
	default:
		// Queue is full as expected
	}
}

func TestClientQueueSize(t *testing.T) {
	client := &Client{
		send: make(chan []byte, clientQueueSize),
	}

	// Test that queue accepts messages up to size
	for i := 0; i < clientQueueSize; i++ {
		select {
		case client.send <- []byte("test"):
			// Message accepted
		default:
			t.Fatal("Queue should accept message")
		}
	}

	// Verify queue is now full
	select {
	case client.send <- []byte("test"):
		t.Fatal("Queue should be full")
	default:
		// Queue is full as expected
	}
}

func TestWorkerRun(t *testing.T) {
	t.Run("Batch size trigger", func(t *testing.T) {
		delivered := make(chan []byte, 1)

		worker := &Worker{
			id:            0,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Minute),
		}

		worker.clients[0] = &Client{
			send: delivered,
		}

		go worker.run()

		// Send enough messages to trigger batch size
		for i := 0; i < batchSize; i++ {
			worker.messages <- []byte("test")
		}

		// Wait for batch delivery
		select {
		case msg := <-delivered:
			assert.Equal(t, []byte("test"), msg)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for batch delivery")
		}

		close(worker.done)
	})

	t.Run("Batch timeout trigger", func(t *testing.T) {
		delivered := make(chan []byte, 1)

		worker := &Worker{
			id:            0,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Minute),
		}
		worker.clients[0] = &Client{
			send: delivered,
		}
		go worker.run()

		worker.messages <- []byte("test1")
		worker.messages <- []byte("test2")

		select {
		case batch := <-delivered:
			assert.Equal(t, []byte("test1"), batch)
		case <-time.After(batchTimeout * 2):
			t.Fatal("Timeout waiting for batch delivery")
		}

		select {
		case batch := <-delivered:
			assert.Equal(t, []byte("test2"), batch)
		case <-time.After(batchTimeout * 2):
			t.Fatal("Timeout waiting for batch delivery")
		}

		close(worker.done)
	})

	t.Run("Cleanup inactive clients", func(t *testing.T) {
		worker := &Worker{
			id:            0,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Millisecond * 50),
		}

		for i := 0; i < 10; i++ {
			worker.clients[uint64(i)] = &Client{
				done: make(chan struct{}),
			}
		}

		go worker.run()

		time.Sleep(100 * time.Millisecond)

		assert.Empty(t, worker.clients, "clients should be empty")
		close(worker.done)
	})

	t.Run("Graceful shutdown", func(t *testing.T) {
		worker := &Worker{
			id:            0,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Minute),
		}

		finishChan := make(chan struct{})
		go func() {
			worker.run()
			close(finishChan)
		}()

		// Send some messages
		worker.messages <- []byte("test1")
		worker.messages <- []byte("test2")

		close(worker.done)

		select {
		case <-finishChan:
			// Worker exited as expected
		case <-time.After(time.Second):
			t.Fatal("Worker did not shut down gracefully")
		}
	})

	t.Run("Message queueing metrics", func(t *testing.T) {
		worker := &Worker{
			id:            0,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Minute),
		}

		queueSizeBefore := worker.metrics.queueSize.Load()

		go worker.run()

		numMessages := 5
		for i := 0; i < numMessages; i++ {
			worker.messages <- []byte("test")
		}

		time.Sleep(500 * time.Millisecond)

		queueSizeAfter := worker.metrics.queueSize.Load()
		assert.Equal(t, queueSizeBefore, queueSizeAfter,
			"Queue size should return to initial value after processing")

		close(worker.done)
	})
}
