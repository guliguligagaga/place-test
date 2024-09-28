package grid

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
	"strconv"
	"time"
	"web"
)

var redisClient *redis.Client

type Update struct {
	Timestamp uint64 `json:"time"`
	Data      string `json:"data"`
}

func Run() {
	kafkaUrl := fmt.Sprintf("%s:%s", os.Getenv("KAFKA_URL"), os.Getenv("KAFKA_PORT"))
	r := kafka.ReaderConfig{
		Brokers: []string{kafkaUrl},
		Topic:   "grid_updates",
		GroupID: "grid-sync-consumer-group",
	}

	c := web.WithKafkaConsumer(r, kafkaConsumer)
	instance := web.MakeServer(c, web.WithRedis)

	redisClient = instance.Redis()

	instance.Run()
}

func kafkaConsumer(r *kafka.Reader) {
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
