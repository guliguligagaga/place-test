package ws

import (
	"backend/logging"
	"github.com/gorilla/websocket"
	"math/rand"
	"sync/atomic"
)

type Clients struct {
	pool       *WorkerPool
	totalConns atomic.Int64
}

func NewClients() *Clients {
	return &Clients{
		pool: NewWorkerPool(),
	}
}

type Client struct {
	ID          uint64
	Conn        *websocket.Conn
	send        chan []byte
	done        chan struct{}
	workerIndex int
}

func (c *Client) Send(data []byte) {
	c.send <- data
}

func (c *Clients) Add(conn *websocket.Conn) *Client {
	clientID := generateClientID()
	workerIndex := int(clientID % uint64(numWorkers))
	worker := c.pool.workers[workerIndex]

	client := &Client{
		ID:          clientID,
		Conn:        conn,
		send:        make(chan []byte, clientQueueSize),
		done:        make(chan struct{}),
		workerIndex: workerIndex,
	}

	worker.clientsLock.Lock()
	if len(worker.clients) >= maxClientsPerWorker {
		worker.clientsLock.Unlock()
		conn.Close()
		return nil
	}
	worker.clients[clientID] = client
	worker.clientsLock.Unlock()

	c.totalConns.Add(1)
	go c.clientWriter(client)
	return client
}

func (c *Clients) Remove(client *Client) {
	worker := c.pool.workers[client.workerIndex]

	worker.clientsLock.Lock()
	delete(worker.clients, client.ID)
	worker.clientsLock.Unlock()

	close(client.done)
	c.totalConns.Add(-1)
}

func (c *Clients) Broadcast(message []byte) {
	for _, worker := range c.pool.workers {
		select {
		case worker.messages <- message:
		default:
			logging.Errorf("Worker queue full, dropping message")
		}
	}
}

func (c *Clients) clientWriter(client *Client) {
	defer client.Conn.Close()

	for {
		select {
		case message := <-client.send:
			err := client.Conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				logging.Errorf("Error writing to client %d: %v", client.ID, err)
				c.Remove(client)
				return
			}
		case <-client.done:
			return
		}
	}
}

var clientCounter uint32

func generateClientID() uint64 {
	counter := atomic.AddUint32(&clientCounter, 1)
	random := rand.Uint32()
	return (uint64(counter) << 32) | uint64(random)
}
