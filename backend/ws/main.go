package ws

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"backend/logging"
	"backend/web"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
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
	redisClient = web.DefaultRedis()

	server := web.NewServer(web.WithRedis(redisClient),
		ginEngine,
		web.WithBackgroundWorker(func(ctx context.Context) {
			consumer(ctx, redisClient)
		}),
	)
	server.RegisterShutdownHook(clients)
	server.RegisterShutdownHook(localCache)

	server.Run()
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

	sendLatestStateAndUpdates(client)
}
func consumer(ctx context.Context, cli redis.UniversalClient) {

	pubsub := cli.Subscribe(context.Background(), "grid_updates_brd")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		data := addMsgType(msgTypeUpdate, []byte(msg.Payload))
		logging.Debugf("got a new message from kafka, broadcasting it")
		clients.Broadcast(data)
		cacheKey := fmt.Sprintf("%s:updates:%d", gridKey, getCurrentEpoch())
		localCache.Update(cacheKey, data)
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
	err = client.sendRaw(data)
	if err != nil {
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
		err = client.sendRaw(data)
		if err != nil {
			logging.Errorf("Client %d queue full when sending state", client.ID)
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
