package ws

import (
	"github.com/gorilla/websocket"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"backend/logging"
)

var numWorkers = runtime.GOMAXPROCS(0) * 2

const (
	workerQueueSize     = 25000
	batchSize           = 500
	batchTimeout        = 50 * time.Millisecond
	maxClientsPerWorker = 1000
	pingInterval        = 30 * time.Second
	writeTimeout        = 100 * time.Millisecond
	readTimeout         = 60 * time.Second
)

type WorkerPool struct {
	workers    []*Worker
	numWorkers int
	metrics    *PoolMetrics
}

type Worker struct {
	id               int
	clients          map[uint64]*Client
	clientsLock      sync.RWMutex
	messages         chan *websocket.PreparedMessage
	done             chan struct{}
	metrics          *WorkerMetrics
	cleanupTicker    *time.Ticker
	dynamicBatchSize int
}

type PoolMetrics struct {
	totalMessages   atomic.Uint64
	droppedMessages atomic.Uint64
	activeClients   atomic.Int32
}

type WorkerMetrics struct {
	queueSize     atomic.Int32
	activeClients atomic.Int32
	batchesSent   atomic.Uint64
	timerResets   atomic.Uint64
}

func NewWorkerPool() *WorkerPool {
	pool := &WorkerPool{
		workers:    make([]*Worker, numWorkers),
		numWorkers: numWorkers,
		metrics:    &PoolMetrics{},
	}

	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			id:               i,
			clients:          make(map[uint64]*Client),
			messages:         make(chan *websocket.PreparedMessage, workerQueueSize),
			done:             make(chan struct{}),
			metrics:          &WorkerMetrics{},
			cleanupTicker:    time.NewTicker(time.Minute),
			dynamicBatchSize: batchSize,
		}
		pool.workers[i] = worker
		go worker.run()
	}

	// Start metrics reporter
	go pool.reportMetrics()

	return pool
}

func (w *Worker) run() {
	batch := make([]*websocket.PreparedMessage, 0, w.dynamicBatchSize)
	timer := time.NewTimer(0) // Create stopped timer
	if !timer.Stop() {
		<-timer.C
	}

	defer func() {
		timer.Stop()
		w.cleanupTicker.Stop()
	}()

	for {
		select {
		case msg := <-w.messages:
			w.metrics.queueSize.Add(1)
			batch = append(batch, msg)

			if len(batch) >= w.dynamicBatchSize {
				w.sendBatch(batch)
				batch = batch[:0]
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			} else if len(batch) == 1 {
				// Start timer only when first message arrives in empty batch
				timer.Reset(batchTimeout)
			}
			w.metrics.queueSize.Add(-1)

		case <-timer.C:
			if len(batch) > 0 {
				w.sendBatch(batch)
				batch = batch[:0]
				w.metrics.batchesSent.Add(1)
			}

		case <-w.cleanupTicker.C:
			w.cleanupInactiveClients()

		case <-w.done:
			return
		}
	}
}

func (p *WorkerPool) reportMetrics() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			total := p.metrics.totalMessages.Load()
			dropped := p.metrics.droppedMessages.Load()
			clients := p.metrics.activeClients.Load()

			logging.Infof("Metrics - Messages: total=%d dropped=%d clients=%d",
				total, dropped, clients)
		}
	}
}

var clientCounter uint32

func generateClientID() uint64 {
	counter := atomic.AddUint32(&clientCounter, 1)
	random := rand.Uint32()
	return (uint64(counter) << 32) | uint64(random)
}
