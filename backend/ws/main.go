package ws

import (
	"backend/logging"
	"backend/web"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

const (
	_            uint8 = iota
	msgTypeState       = 1 << iota
	msgTypeUpdate

	redisRetryAttempts = 3
	redisRetryDelay    = 500 * time.Millisecond
	kafkaMaxRetries    = 3
	kafkaRetryDelay    = 1 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	gridKey = os.Getenv("REDIS_GRID_KEY")

	clients     = NewClients()
	redisClient redis.UniversalClient
	localCache  *Cache
)

func Run() {

	ginEngine := web.WithGinEngine(func(r *gin.Engine) {
		r.GET("/ws", handleWebSocket)
	})

	kafkaUrl := fmt.Sprintf("%s:%s", os.Getenv("KAFKA_URL"), os.Getenv("KAFKA_PORT"))
	r := kafka.ReaderConfig{
		Brokers:        []string{kafkaUrl},
		Topic:          "grid_updates",
		GroupID:        os.Getenv("POD_NAME"),
		MinBytes:       10e3,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,

		// Enable reading in batches
		ReadBatchTimeout: 100 * time.Millisecond,
		MaxAttempts:      kafkaMaxRetries,
	}
	localCache = NewCache(5)
	go localCache.runCleanup()
	consumer := web.WithKafkaConsumer(r, kafkaConsumer)

	instance := web.MakeServer(ginEngine, web.WithDefaultRedis, consumer)
	instance.AddCloseOnExit(clients)
	instance.AddCloseOnExit(localCache)
	redisClient = instance.Redis()

	instance.Run()
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Errorf("Upgrade error: %v", err)
		return
	}

	client := clients.Add(conn)
	if client == nil {
		logging.Errorf("Failed to add client - worker pool full")
		conn.Close()
		return
	}

	go sendLatestStateAndUpdates(client)
}

func kafkaConsumer(r *kafka.Reader) {
	ctx := context.Background()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if err == kafka.ErrGroupClosed {
				return
			}
			logging.Errorf("Kafka batch read error: %v", err)
			time.Sleep(kafkaRetryDelay)
			continue
		}

		data := addMsgType(msgTypeUpdate, m.Value)
		logging.Debugf("got a new message from kafka, broadcasting it")
		clients.Broadcast(data)
		cacheKey := fmt.Sprintf("%s:updates:%d", gridKey, getCurrentEpoch())
		localCache.Update(cacheKey, data)

		if err := r.CommitMessages(ctx, m); err != nil {
			logging.Errorf("Failed to commit messages: %v", err)
		}
	}
}

func getCurrentEpoch() int64 {
	epoch := time.Now().UnixMilli() / 60_000
	return epoch
}

func sendLatestStateAndUpdates(client *Client) {
	ctx := context.Background()
	epoch := getCurrentEpoch()

	var state string
	var err error
	for i := 0; i < redisRetryAttempts; i++ {
		state, err = redisClient.Get(ctx, gridKey).Result()
		if err == nil {
			break
		}
		if err == redis.Nil {
			logging.Infof("No state found in Redis")
			return
		}
		logging.Infof("Retry %d: Error getting state: %v", i+1, err)
		time.Sleep(redisRetryDelay)
	}

	if err != nil {
		logging.Errorf("Failed to get state after %d attempts: %v", redisRetryAttempts, err)
		return
	}

	data := addMsgType(msgTypeState, []byte(state))
	select {
	case client.send <- data:
	case <-client.done:
		return
	default:
		logging.Errorf("Client %d queue full when sending state", client.ID)
		return
	}

	updates := make([][]byte, 0)
	cacheKey := fmt.Sprintf("%s:updates:%d", gridKey, epoch)
	if cachedUpdates, ok := localCache.Get(cacheKey); ok {
		updates = append(updates, cachedUpdates...)
	}

	// If no cached updates, try Redis
	if len(updates) == 0 {
		redisUpdates, err := redisClient.ZRangeByScore(ctx, cacheKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: "+inf",
		}).Result()

		if err != nil && err != redis.Nil {
			logging.Errorf("Error getting updates: %v", err)
		} else {
			for _, update := range redisUpdates {
				updates = append(updates, []byte(update))
			}
		}
	}

	for _, update := range updates {
		data = addMsgType(msgTypeUpdate, update)
		select {
		case client.send <- data:
		case <-client.done:
			return
		default:
			logging.Errorf("Client %d queue full when sending updates", client.ID)
			return
		}
	}
}

func addMsgType(msgType uint8, msg []byte) []byte {
	result := make([]byte, len(msg)+1)
	result[0] = msgType
	copy(result[1:], msg)
	return result
}

func init() {
	requiredEnvVars := []string{
		"KAFKA_URL",
		"KAFKA_PORT",
		"REDIS_GRID_KEY",
	}

	for _, env := range requiredEnvVars {
		if os.Getenv(env) == "" {
			panic(fmt.Sprintf("Required environment variable %s is not set", env))
		}
	}
}
