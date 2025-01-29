package ws

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"backend/logging"
	"github.com/gorilla/websocket"
)

var numWorkers = runtime.NumCPU()

const (
	workerQueueSize     = 10000
	clientQueueSize     = 256
	batchSize           = 100
	batchTimeout        = 200 * time.Millisecond
	maxClientsPerWorker = 5000
	pingInterval        = 30 * time.Second
	writeTimeout        = 10 * time.Second
	readTimeout         = 60 * time.Second
)

type WorkerPool struct {
	workers    []*Worker
	numWorkers int
	metrics    *PoolMetrics
}

type Worker struct {
	id            int
	clients       map[uint64]*Client
	clientsLock   sync.RWMutex
	messages      chan []byte
	done          chan struct{}
	metrics       *WorkerMetrics
	cleanupTicker *time.Ticker
}

type Client struct {
	ID       uint64
	Conn     *websocket.Conn
	send     chan []byte
	done     chan struct{}
	worker   *Worker
	lastPing atomic.Int64
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
			id:            i,
			clients:       make(map[uint64]*Client),
			messages:      make(chan []byte, workerQueueSize),
			done:          make(chan struct{}),
			metrics:       &WorkerMetrics{},
			cleanupTicker: time.NewTicker(time.Minute),
		}
		pool.workers[i] = worker
		go worker.run()
	}

	// Start metrics reporter
	go pool.reportMetrics()

	return pool
}

func (w *Worker) run() {
	batch := make([][]byte, 0, batchSize)
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

			if len(batch) >= batchSize {
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

			//for _, w := range p.workers {
			//	logging.Infof("Worker %d - Queue=%d Clients=%d",
			//		w.id,
			//		w.metrics.queueSize.Load(),
			//		w.metrics.activeClients.Load())
			//}
		}
	}
}

var clientCounter uint32

func generateClientID() uint64 {
	counter := atomic.AddUint32(&clientCounter, 1)
	random := rand.Uint32()
	return (uint64(counter) << 32) | uint64(random)
}
