package ws

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"web"
)

type Clients struct {
	sync.RWMutex
	clients map[int64]*websocket.Conn
}

func (c *Clients) Close() error {
	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	for _, conn := range c.clients {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing connection:", err)
		}
	}
	return nil
}

var clients = &Clients{
	clients: make(map[int64]*websocket.Conn),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var redisClient *redis.Client

func Run() {
	ginEngine := web.WithGinEngine(func(r *gin.Engine) {
		r.GET("/ws", handleWebSocket)
	})
	kafkaUrl := fmt.Sprintf("%s:%s", os.Getenv("KAFKA_URL"), os.Getenv("KAFKA_PORT"))
	r := kafka.ReaderConfig{
		Brokers:  []string{kafkaUrl},
		Topic:    "grid_updates",
		GroupID:  "ws",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  500 * time.Millisecond,
	}
	consumer := web.WithKafkaConsumer(r, kafkaConsumer)

	instance := web.MakeServer(ginEngine, web.WithRedis, consumer)
	instance.AddCloseOnExit(clients)
	redisClient = instance.Redis()
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
		_, _, err = conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		//log.Printf("Received message from client %s: %s", clientID, msg)
	}

	clients.Lock()
	delete(clients.clients, clientID)
	clients.Unlock()
}

func sendLatestStateAndUpdates(conn *websocket.Conn) {
	ctx := context.Background()

	epoch := time.Now().UnixMilli() / 60_000
	// Get latest epoch
	latestEpoch, err := redisClient.Get(ctx, "latest_epoch").Result()
	if err != nil {
		log.Println("Error getting latest epoch:", err)
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte(latestEpoch))
	if err != nil {
		log.Println("Error sending latest epoch:", err)
	}

	// get latest state
	int64s, err := redisClient.BitField(ctx, "grid", "get", "u4", "0", "0").Result()
	if err != nil {
		log.Println("Error getting latest state:", err)
	}
	buf := new(bytes.Buffer)
	for _, i := range int64s {
		err = binary.Write(buf, binary.LittleEndian, i)
		if err != nil {
			log.Println("Error writing to buffer:", err)
			break
		}
	}
	err = conn.WriteMessage(websocket.TextMessage, buf.Bytes())
	if err != nil {
		log.Println("Error sending latest state:", err)
	}

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

func kafkaConsumer(r *kafka.Reader) {
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
}
