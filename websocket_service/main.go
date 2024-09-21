package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

type Clients struct {
	sync.RWMutex
	clients map[string]*websocket.Conn
}

var clients = Clients{
	clients: make(map[string]*websocket.Conn),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var redisClient *redis.Client

type Update struct {
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
}

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	server := &http.Server{Addr: ":8082"}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go kafkaConsumer([]string{"localhost:9092"}, "grid_updates")

	<-ctx.Done()
	log.Println("Shutdown signal received")

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("%d", len(clients.clients)+1)
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

func kafkaConsumer(brokers []string, topic string) {
	ctx := context.Background()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "websocket_service",
	})

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Println("Kafka read error:", err)
			return
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		// Store update in Redis
		update := Update{
			Timestamp: time.Now().UnixNano(),
			Data:      string(m.Value),
		}
		updateJSON, _ := json.Marshal(update)
		epoch := time.Now().UnixMilli() / 60_000
		err = redisClient.ZAdd(ctx, fmt.Sprintf("updates:%d", epoch), &redis.Z{
			Score:  float64(update.Timestamp),
			Member: string(updateJSON),
		}).Err()
		if err != nil {
			log.Println("Error storing update in Redis:", err)
		}

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
}
