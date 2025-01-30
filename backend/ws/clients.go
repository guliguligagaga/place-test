package ws

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"backend/logging"
	"github.com/gorilla/websocket"
)

const (
	pongWait     = 60 * time.Second
	pingInterval = (pongWait * 9) / 10
	writeTimeout = 100 * time.Millisecond
	readTimeout  = 1 * time.Second
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
	//client.Conn.SetPingHandler(func(string) error {
	//	if err := client.Conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(writeTimeout)); err != nil {
	//		return err
	//	}
	//	return nil
	//})
	defer func() {
		c.remove(client)
	}()
	client.Conn.SetPongHandler(func(s string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		client.lastPing.Store(time.Now().UnixNano())
		return nil
	})

	_ = client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Errorf("unexpected close error: %v", err)
			}

			break
		}
	}
}

func (c *Clients) Broadcast(message []byte) {
	prepMsg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, message)
	if err != nil {
		logging.Errorf("failed to create perp msg %v", err)
		return
	}

	c.pool.Range(func(key, value any) bool {
		cli := value.(*Client)
		select {
		case cli.writePipe <- prepMsg:
		default:
			logging.Debugf("client is full, closing it ")
			cli.done <- struct{}{}
		}
		return true
	})

}

func (c *Clients) Close() error {
	c.pingTicker.Stop()
	c.pool.Range(func(key, value any) bool {
		value.(*Client).sendCloseMsg()
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
			err := client.Conn.WritePreparedMessage(msg)
			if err != nil {
				logging.Errorf("Failed to send msg to client %d: %v", client.ID, err)
				return
			}
		case <-c.pingTicker.C:
			err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				if !websocket.IsCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure) {
					logging.Errorf("Failed to send ping to client %d: %v", client.ID, err)
				}
				c.remove(client)
				return
			}
		case <-c.cleanupTicker.C:
			c.cleanupInactiveClients()
		case <-client.done:
			return
		}
	}
}

func (c *Clients) cleanupInactiveClients() {
	threshold := time.Now().Add(-2 * time.Minute).UnixNano()
	inactiveCount := 0

	c.pool.Range(func(key, value any) bool {
		client := value.(*Client)
		if client.lastPing.Load() < threshold {
			c.remove(client)
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

func (c *Clients) remove(cli *Client) {
	c.pool.Delete(cli.ID)
	close(cli.writePipe)
	cli.Conn.Close()
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

func (c *Client) sendCloseMsg() {
	close(c.writePipe)
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err := c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logging.Errorf("Failed to send close to client %d: %v", c.ID, err)
	}
	logging.Infof("Closed client %d", c.ID)
}

var clientCounter uint32

func generateClientID() uint64 {
	counter := atomic.AddUint32(&clientCounter, 1)
	random := rand.Uint32()
	return (uint64(counter) << 32) | uint64(random)
}
