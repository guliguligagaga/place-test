package ws

import (
	"backend/logging"
	"sync"
	"time"
)

type WorkerPool struct {
	workers    []*Worker
	numWorkers int
}

type Worker struct {
	clients     map[uint64]*Client
	clientsLock sync.RWMutex
	messages    chan []byte
	done        chan struct{}
}

const (
	numWorkers          = 16    // Number of worker goroutines
	workerQueueSize     = 10000 // Size of worker message queue
	clientQueueSize     = 64    // Size of individual client queue
	batchSize           = 100   // Number of messages to batch
	batchTimeout        = 50 * time.Millisecond
	maxClientsPerWorker = 5000 // Maximum clients per worker
)

func NewWorkerPool() *WorkerPool {
	pool := &WorkerPool{
		workers:    make([]*Worker, numWorkers),
		numWorkers: numWorkers,
	}

	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			clients:  make(map[uint64]*Client),
			messages: make(chan []byte, workerQueueSize),
			done:     make(chan struct{}),
		}
		pool.workers[i] = worker
		go worker.run()
	}

	return pool
}

func (w *Worker) run() {
	batch := make([][]byte, 0, batchSize)
	timer := time.NewTimer(batchTimeout)

	for {
		select {
		case msg := <-w.messages:
			batch = append(batch, msg)
			if len(batch) >= batchSize {
				w.sendBatch(batch)
				batch = batch[:0]
				timer.Reset(batchTimeout)
			}
		case <-timer.C:
			if len(batch) > 0 {
				w.sendBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(batchTimeout)
		case <-w.done:
			return
		}
	}
}

func (w *Worker) sendBatch(messages [][]byte) {
	w.clientsLock.RLock()
	defer w.clientsLock.RUnlock()

	for _, client := range w.clients {
		select {
		case <-client.done:
			continue
		default:
			for _, msg := range messages {
				select {
				case client.send <- msg:
				default:
					// Client queue full, log and continue
					logging.Errorf("Client %d queue full", client.ID)
				}
			}
		}
	}
}

func (w *Worker) close() {
	//stop processing messages
	close(w.done)

	w.clientsLock.Lock()
	for _, client := range w.clients {
		close(client.done)
	}
	w.clientsLock.Unlock()
}
