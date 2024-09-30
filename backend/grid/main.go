package grid

import (
	"backend/logging"
	"backend/web"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"os"
	"strconv"
	"time"
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
	instance := web.MakeServer(c, web.WithRedis, web.WithGinEngine(func(r *gin.Engine) {}))

	redisClient = instance.Redis()

	instance.Run()
}

type data struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Color int `json:"color"`
}

func kafkaConsumer(r *kafka.Reader) {
	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			logging.Errorf("Kafka read error:%v", err)
			continue
		}

		// Store update in Redis
		updateTime := time.Now().UnixMilli()
		updateEpoch := updateTime / 60_000
		err = redisClient.ZAdd(ctx, fmt.Sprintf("updates:%d", updateEpoch), &redis.Z{
			Score:  float64(m.Time.UnixMilli()),
			Member: string(m.Value),
		}).Err()
		if err != nil {
			logging.Errorf("Error storing update in Redis:%v", err)
		}

		err = redisClient.Set(ctx, "latest_epoch", strconv.FormatInt(updateEpoch, 10), 0).Err()
		if err != nil {
			logging.Errorf("Error updating latest state in Redis:%v", err)
		}
		d := data{}
		_ = json.Unmarshal(m.Value, &d)
		gridKey := "grid"
		byteIndex := (d.Y*100 + d.X) / 2
		isUpperNibble := d.X%2 == 0

		var offset int64
		if isUpperNibble {
			offset = int64(byteIndex * 8)
		} else {
			offset = int64(byteIndex*8 + 4)
		}
		_, err = redisClient.BitField(ctx, gridKey, "SET", "u4", offset, d.Color).Result()
		if err != nil {
			logging.Errorf("Error updating grid status in Redis:%v", err)
		}
		err = r.CommitMessages(ctx, m)
		if err != nil {
			logging.Errorf("Error committing message:%v", err)
		}
	}
}
