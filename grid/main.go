package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	updatesTopic     = "grid-updates"
	kafkaGroupID     = "grid-sync-consumer-group"
	redisKeyPrefix   = "grid:update:"
	redisCacheWindow = time.Hour * 24 // Cache updates for 24 hours
)

var redisClient *redis.Client

type Update struct {
	Timestamp uint64 `json:"time"`
	Data      string `json:"data"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go kafkaConsumer(ctx, []string{"kafka:29092"}, "grid_updates")
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	<-ctx.Done()
	log.Println("Shutdown signal received")
	log.Println("Server exited properly")
}

func kafkaConsumer(ctx context.Context, brokers []string, topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "grid-sync-consumer-group",
	})

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Println("Kafka read error:", err)
			return
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		// Store update in Redis
		updateTime := time.Now().UnixMilli()
		updateEpoch := updateTime / 60_000
		update := Update{
			Timestamp: uint64(updateTime),
			Data:      string(m.Value),
		}
		updateJSON, _ := json.Marshal(update)
		err = redisClient.ZAdd(ctx, fmt.Sprintf("updates:%d", updateEpoch), &redis.Z{
			Score:  float64(update.Timestamp),
			Member: string(updateJSON),
		}).Err()
		if err != nil {
			log.Println("Error storing update in Redis:", err)
		}

		err = redisClient.Set(ctx, "latest_epoch", strconv.FormatInt(updateEpoch, 10), 0).Err()
		if err != nil {
			log.Println("Error updating latest state in Redis:", err)
		}

		err = r.CommitMessages(ctx, m)
		if err != nil {
			log.Println("Error committing message:", err)
		}
	}
}
