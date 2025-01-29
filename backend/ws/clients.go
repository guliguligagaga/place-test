package ws

import (
	"sync/atomic"
	"time"

	"backend/logging"
	"github.com/gorilla/websocket"
)

type Clients struct {
	pool       *WorkerPool
	totalConns atomic.Int64

	pingTicker *time.Ticker
	pingMsg    *websocket.PreparedMessage
}

type Client struct {
	ID        uint64
	Conn      *websocket.Conn
	writePipe chan *websocket.PreparedMessage
	done      chan struct{}
	worker    *Worker
	lastPing  atomic.Int64
}

func NewClients() *Clients {
	pingMsg, _ := websocket.NewPreparedMessage(websocket.PingMessage, []byte{})
	return &Clients{
		pool:       NewWorkerPool(),
		pingTicker: time.NewTicker(pingInterval),
		pingMsg:    pingMsg,
	}
}

func (c *Clients) Add(conn *websocket.Conn) *Client {
	// Find least loaded worker
	var selectedWorker *Worker
	minClients := int32(maxClientsPerWorker)

	for _, worker := range c.pool.workers {
		numClients := worker.metrics.activeClients.Load()
		if numClients < minClients {
			minClients = numClients
			selectedWorker = worker
		}
	}

	if selectedWorker == nil || minClients >= maxClientsPerWorker {
		logging.Errorf("failed to choose worker for client")
		return nil
	}
	clientID := generateClientID()
	logging.Debugf("client %d assigned to worker %d", clientID, selectedWorker.id)
	client := &Client{
		ID:        clientID,
		Conn:      conn,
		writePipe: make(chan *websocket.PreparedMessage, 256),
		done:      make(chan struct{}),
		worker:    selectedWorker,
	}
	client.lastPing.Store(time.Now().UnixNano())

	selectedWorker.clientsLock.Lock()
	selectedWorker.clients[clientID] = client
	selectedWorker.clientsLock.Unlock()
	selectedWorker.metrics.activeClients.Add(1)

	c.totalConns.Add(1)
	c.pool.metrics.activeClients.Add(1)

	go c.writePump(client)
	go c.readPump(client)

	return client
}

func (c *Clients) readPump(client *Client) {
	client.Conn.SetReadLimit(512) // Small limit since we don't expect client messages
	_ = client.Conn.SetReadDeadline(time.Now().Add(readTimeout))

	client.Conn.SetPongHandler(func(string) error {
		_ = client.Conn.SetReadDeadline(time.Now().Add(readTimeout))
		client.lastPing.Store(time.Now().UnixNano())
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				logging.Errorf("read error: %v", err)
			}
			break
		}
	}

	c.Remove(client)
}

func (c *Clients) Remove(client *Client) {
	select {
	case <-client.done:
		return
	default:
		close(client.done)

		client.worker.clientsLock.Lock()
		delete(client.worker.clients, client.ID)
		client.worker.clientsLock.Unlock()
		client.worker.metrics.activeClients.Add(-1)
		c.totalConns.Add(-1)
		c.pool.metrics.activeClients.Add(-1)
		client.Conn.Close()
	}
}

func (c *Clients) Broadcast(message []byte) {
	c.pool.metrics.totalMessages.Add(1)
	prepMsg, _ := websocket.NewPreparedMessage(websocket.BinaryMessage, message)

	for _, worker := range c.pool.workers {
		select {
		case worker.messages <- prepMsg:
		default:
			c.pool.metrics.droppedMessages.Add(1)
			logging.Errorf("Worker %d queue full, dropping message", worker.id)
		}
	}
}

func (c *Clients) Close() error {
	for _, worker := range c.pool.workers {
		worker.done <- struct{}{}
	}
	return nil
}

func (c *Clients) writePump(client *Client) {
	for {
		select {
		case msg, ok := <-client.writePipe:
			client.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := client.Conn.WritePreparedMessage(msg)
			if err != nil {
				logging.Errorf("Failed to send msg to client %d: %v", client.ID, err)
				return
			}
		case <-c.pingTicker.C:
			client.writePipe <- clients.pingMsg
			logging.Debugf("Sent ping to client %d", client.ID)

		case <-client.done:
			c.pingTicker.Stop()
			client.Conn.Close()
			logging.Infof("Closed writer for client %d", client.ID)
			return
		}
	}
}

func (w *Worker) cleanupInactiveClients() {
	threshold := time.Now().Add(-2 * time.Minute).UnixNano()
	inactiveCount := 0

	w.clientsLock.Lock()
	for id, client := range w.clients {
		if client.lastPing.Load() < threshold {
			delete(w.clients, id)
			close(client.done)
			w.metrics.activeClients.Add(-1)
			inactiveCount++
			logging.Infof("Cleaned up inactive client %d (last ping: %v)",
				id, time.Unix(0, client.lastPing.Load()))
		}
	}
	w.clientsLock.Unlock()

	if inactiveCount > 0 {
		logging.Infof("Worker %d cleaned up %d inactive clients", w.id, inactiveCount)
	}
}

func (w *Worker) sendBatch(messages []*websocket.PreparedMessage) {
	logging.Debugf("sending a batch size %d from worker %d", len(messages), w.id)
	w.clientsLock.RLock()
	clientCount := len(w.clients)
	clientsList := make([]*Client, 0, clientCount)
	for _, client := range w.clients {
		select {
		case <-client.done:
			continue
		default:
			clientsList = append(clientsList, client)
		}
	}
	w.clientsLock.RUnlock()

	logging.Debugf("Worker %d sending batch of %d messages to %d clients",
		w.id, len(messages), len(clientsList))

	for _, msg := range messages {
		for _, client := range clientsList {
			client.writePipe <- msg
		}
	}
}

func (c *Client) sendRaw(message []byte) error {
	err := c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		logging.Errorf("Failed to set deadline timeout %d: %v", c.ID, err)
		return err
	}
	if err = c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		logging.Errorf("Failed writing to client %d: %v", c.ID, err)
		return err
	}
	return nil
}
