package ws

import (
	"backend/logging"
	"github.com/gorilla/websocket"
	"math/rand"
	"sync"
	"sync/atomic"
)

type Clients struct {
	sync.RWMutex
	clients map[uint64]*websocket.Conn
}

func (c *Clients) Close() error {
	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	for _, conn := range c.clients {
		err := conn.Close()
		if err != nil {
			logging.Errorf("Error closing connection: %v", err)
		}
	}
	return nil
}

var clients = &Clients{
	clients: make(map[uint64]*websocket.Conn),
}

var clientCounter uint32

func generateClientID() uint64 {
	counter := atomic.AddUint32(&clientCounter, 1)
	random := rand.Uint32()
	return (uint64(counter) << 32) | uint64(random)
}
