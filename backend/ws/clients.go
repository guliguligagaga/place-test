package ws

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"backend/logging"
	"github.com/gorilla/websocket"
)

type Clients struct {
	pool       *sync.Map
	totalConns atomic.Int64

	pingTicker    *time.Ticker
	cleanupTicker *time.Ticker
	pingMsg       *websocket.PreparedMessage
}

type Client struct {
	serverCtx context.Context
	ID        uint64
	Conn      *websocket.Conn
	writePipe chan *websocket.PreparedMessage
	done      chan struct{}
	lastPing  atomic.Int64
}

func NewClients() *Clients {
	pingMsg, _ := websocket.NewPreparedMessage(websocket.PingMessage, []byte{})
	return &Clients{
		pool:          &sync.Map{},
		cleanupTicker: time.NewTicker(time.Minute),
		pingTicker:    time.NewTicker(pingInterval),
		pingMsg:       pingMsg,
	}
}

func (c *Clients) Add(conn *websocket.Conn) *Client {
	clientID := generateClientID()
	client := &Client{
		ID:        clientID,
		Conn:      conn,
		writePipe: make(chan *websocket.PreparedMessage, 256),
		done:      make(chan struct{}),
	}
	client.lastPing.Store(time.Now().UnixNano())
	c.pool.Store(clientID, client)

	c.totalConns.Add(1)

	go c.writePump(client)
	go c.readPump(client)

	return client
}

func (c *Clients) readPump(client *Client) {
	client.Conn.SetReadLimit(512) // Small limit since we don't expect client messages

	client.Conn.SetPongHandler(func(string) error {
		_ = client.Conn.SetReadDeadline(time.Now().Add(readTimeout))
		client.lastPing.Store(time.Now().UnixNano())
		return nil
	})

	for {
		_ = client.Conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("conneaction read error: %v", err)
			}
			break
		}
	}
}

func (c *Clients) Broadcast(message []byte) {
	prepMsg, _ := websocket.NewPreparedMessage(websocket.BinaryMessage, message)

	c.pool.Range(func(key, value any) bool {
		cli := value.(*Client)
		select {
		case cli.writePipe <- prepMsg:
		default:
			cli.done <- struct{}{}
		}
		return true
	})

}

func (c *Clients) Close() error {
	c.pingTicker.Stop()
	c.pool.Range(func(key, value any) bool {
		value.(*Client).close()
		return true
	})
	return nil
}

func (c *Clients) writePump(client *Client) {
	for {
		select {
		case msg, ok := <-client.writePipe:
			client.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				return
			}
			logging.Debugf("%v", msg)
			err := client.Conn.WritePreparedMessage(msg)
			if err != nil {
				logging.Errorf("Failed to send msg to client %d: %v", client.ID, err)
				return
			}
		case <-c.pingTicker.C:
			client.writePipe <- clients.pingMsg
			logging.Debugf("Sent ping to client %d", client.ID)
		case <-c.cleanupTicker.C:
			c.cleanupInactiveClients()
		}
	}
}

func (c *Clients) cleanupInactiveClients() {
	threshold := time.Now().Add(-2 * time.Minute).UnixNano()
	inactiveCount := 0

	c.pool.Range(func(key, value any) bool {
		client := value.(*Client)
		if client.lastPing.Load() < threshold {
			c.pool.Delete(key)
			client.close()
			inactiveCount++
			logging.Infof("Cleaned up inactive client %d (last ping: %v)",
				client.ID, time.Unix(0, client.lastPing.Load()))
		}
		return true
	})

	if inactiveCount > 0 {
		logging.Infof("cleaned up %d inactive clients", inactiveCount)
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

func (c *Client) close() {
	close(c.writePipe)
	logging.Debugf("closing client %d", c.ID)
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err := c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logging.Errorf("Failed to send close to client %d: %v", c.ID, err)
	}
	logging.Infof("Closed writer for client %d", c.ID)
}
