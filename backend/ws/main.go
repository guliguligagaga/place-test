package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"server"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

type Clients struct {
	sync.RWMutex
	clients map[int64]*websocket.Conn
}

var clients = Clients{
	clients: make(map[int64]*websocket.Conn),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
}

func main() {
	r := gin.Default()
	r.GET("/ws", handleWebSocket)

	s := &http.Server{
		Addr:    "0.0.0.0:8082",
		Handler: r,
	}
	instance := server.NewInstance()
	instance.AddOnStart(func() error {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{"kafka:29092"},
			Topic:    "grid_updates",
			GroupID:  "ws",
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
			MaxWait:  500 * time.Millisecond,
		})
		instance.AddCloseOnExit(r)
		return kafkaConsumer(r)
	})

	instance.AddOnStart(s.ListenAndServe)
	instance.AddCloseOnExit(s)
	instance.AddCloseOnExit(redisClient)
	instance.Run()
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientID := time.Now().UnixNano()
	clients.Lock()
	clients.clients[clientID] = conn
	clients.Unlock()
	sendLatestStateAndUpdates(conn)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received message from client %s: %s", clientID, msg)
	}

	clients.Lock()
	delete(clients.clients, clientID)
	clients.Unlock()
}

func sendLatestStateAndUpdates(conn *websocket.Conn) {
	ctx := context.Background()

	epoch := time.Now().UnixMilli() / 60_000
	// Get updates since last connection
	updates, err := redisClient.ZRangeByScore(ctx, fmt.Sprintf("updates:%d", epoch), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if err != nil {
		log.Println("Error getting updates:", err)
	}

	for _, update := range updates {
		err = conn.WriteMessage(websocket.TextMessage, []byte(update))
		if err != nil {
			log.Println("Error sending update:", err)
		}
	}
}

func kafkaConsumer(r *kafka.Reader) error {

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		// Broadcast to connected clients
		clients.RLock()
		for _, conn := range clients.clients {
			err := conn.WriteMessage(websocket.TextMessage, m.Value)
			if err != nil {
				log.Println("Write error:", err)
			}
		}
		clients.RUnlock()
	}

	return nil
}
