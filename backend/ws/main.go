package ws

import (
	"backend/logging"
	"backend/web"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	_            uint8 = iota
	msgTypeState       = 1 << iota
	msgTypeUpdate

	redisRetryAttempts = 3
	redisRetryDelay    = 500 * time.Millisecond
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
	localCache = NewCache(5)
	go localCache.runCleanup()
	ctx, cancelFunc := context.WithCancel(context.Background())

	instance := web.MakeServer(ginEngine, web.WithDefaultRedis)
	instance.AddCloseOnExit(clients)
	instance.AddCloseOnExit(localCache)
	redisClient = instance.Redis()
	instance.AddOnExit(cancelFunc)
	go func() {
		err := consumer(ctx, redisClient)
		panic("failed to create consumer " + err.Error())
	}()
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
func consumer(ctx context.Context, cli redis.UniversalClient) error {
	err := cli.XGroupCreate(context.Background(), "grid_updates", "websocket-group", "0").Err()
	if err != nil && !strings.Contains(fmt.Sprint(err), "BUSYGROUP") {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := cli.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    "websocket-group",
			Consumer: os.Getenv("POD_NAME"),
			Streams:  []string{"grid_updates", ">"},
			Count:    1,
			Block:    time.Second * 1,
		}).Result()

		if err != nil && err != redis.Nil {
			return err
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				var messageValue string

				if value, ok := message.Values["values"].(string); ok {
					messageValue = value
				}
				if messageValue == "" {
					continue
				}
				data := addMsgType(msgTypeUpdate, []byte(messageValue))
				logging.Debugf("got a new message from kafka, broadcasting it")
				clients.Broadcast(data)
				cacheKey := fmt.Sprintf("%s:updates:%d", gridKey, getCurrentEpoch())
				localCache.Update(cacheKey, data)

				cli.XAck(ctx, "grid_updates", "grid-sync-consumer-group", message.ID)
			}
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
