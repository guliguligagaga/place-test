package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"server"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
)

var redisClient *redis.Client

type Update struct {
	Timestamp uint64 `json:"time"`
	Data      string `json:"data"`
}

func main() {

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"kafka:29092"},
		Topic:   "grid_updates",
		GroupID: "grid-sync-consumer-group",
	})
	instance := server.NewInstance()
	instance.AddOnStart(func() error {
		return kafkaConsumer(r)
	})
	instance.AddCloseOnExit(r)

	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	instance.AddCloseOnExit(redisClient)
	instance.Run()
}

func kafkaConsumer(r *kafka.Reader) error {
	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
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
