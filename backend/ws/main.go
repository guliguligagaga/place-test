package ws

import (
	"backend/logging"
	"backend/web"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"log"
	"net/http"
	"os"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var redisClient redis.UniversalClient

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

	instance := web.MakeServer(ginEngine, web.WithDefaultRedis, consumer)
	instance.AddCloseOnExit(clients)
	redisClient = instance.Redis()
	instance.Run()
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Errorf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	clientID := generateClientID()
	clients.Lock()
	clients.clients[clientID] = conn
	clients.Unlock()
	sendLatestStateAndUpdates(conn)

	for {
		_, _, err = conn.ReadMessage()
		if err != nil {
			logging.Errorf("Read error: %v", err)
			break
		}
		//logging.Printf("Received message from client %s: %s", clientID, msg)
	}

	clients.Lock()
	delete(clients.clients, clientID)
	clients.Unlock()
}

const (
	_            uint8 = iota
	msgTypeState       = 1 << iota
	msgTypeUpdate
)

func addMsgType(msgType uint8, msg []byte) []byte {
	return append([]byte{msgType}, msg...)
}

func sendLatestStateAndUpdates(conn *websocket.Conn) {
	ctx := context.Background()
	epoch := time.Now().UnixMilli() / 60_000

	res, err := redisClient.Get(ctx, "grid").Result()
	if err != nil {
		logging.Errorf("Error getting latest state:%v", err)
	}
	data := addMsgType(msgTypeState, []byte(res))
	err = conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		logging.Errorf("Error sending latest state:%v", err)
	}

	updates, err := redisClient.ZRangeByScore(ctx, fmt.Sprintf("updates:%d", epoch), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if err != nil {
		logging.Errorf("Error getting updates:%v", err)
	}

	for _, update := range updates {
		data = addMsgType(msgTypeUpdate, []byte(update))
		err = conn.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			logging.Errorf("Error sending update:%v", err)
		}
	}
}

func kafkaConsumer(r *kafka.Reader) {
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			logging.Errorf("Kafka read error:%v", err)
			continue
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		clients.RLock()
		for _, conn := range clients.clients {
			err = conn.WriteMessage(websocket.BinaryMessage, m.Value)
			if err != nil {
				logging.Errorf("Write error:%v", err)
			}
		}
		clients.RUnlock()
	}
}
