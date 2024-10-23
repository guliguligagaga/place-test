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
	"net/http"
	"os"
	"time"
)

const (
	_            uint8 = iota
	msgTypeState       = 1 << iota
	msgTypeUpdate
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var redisClient redis.UniversalClient
var clients = NewClients()
var gridKey = os.Getenv("REDIS_GRID_KEY")

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

func addMsgType(msgType uint8, msg []byte) []byte {
	return append([]byte{msgType}, msg...)
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Errorf("Upgrade error: %v", err)
		return
	}

	client := clients.Add(conn)
	if client == nil {
		conn.Close()
		return
	}

	go sendLatestStateAndUpdates(client)

	// Read pump
	for {
		_, _, err = conn.ReadMessage()
		if err != nil {
			break
		}
	}

	clients.Remove(client)
}

func kafkaConsumer(r *kafka.Reader) {
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			logging.Errorf("Kafka read error:%v", err)
			continue
		}

		data := addMsgType(msgTypeUpdate, m.Value)
		clients.Broadcast(data)
	}
}

func sendLatestStateAndUpdates(client *Client) {
	ctx := context.Background()
	epoch := time.Now().UnixMilli() / 60_000

	res, err := redisClient.Get(ctx, gridKey).Result()
	if err != nil {
		logging.Errorf("Error getting latest state:%v", err)
		return
	}

	data := addMsgType(msgTypeState, []byte(res))
	client.Send(data)

	updates, err := redisClient.ZRangeByScore(ctx, fmt.Sprintf("%s:updates:%d", gridKey, epoch), &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if err != nil {
		logging.Errorf("Error getting updates:%v", err)
		return
	}

	for _, update := range updates {
		data = addMsgType(msgTypeUpdate, []byte(update))
		client.Send(data)
	}
}
