package grid

import (
	"backend/web"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"os"
)

const (
	size             = 100
	UpdatesKeyPrefix = "updates"
	LatestEpochKey   = "latest_epoch"
)

type Update struct {
	Timestamp uint64 `json:"time"`
	Data      string `json:"data"`
}

type KafkaConfig struct {
	URL     string
	Port    string
	Topic   string
	GroupID string
}

func Run() {
	kafkaConfig := KafkaConfig{
		URL:     os.Getenv("KAFKA_URL"),
		Port:    os.Getenv("KAFKA_PORT"),
		Topic:   "grid_updates",
		GroupID: "grid-sync-consumer-group",
	}

	r := kafka.ReaderConfig{
		Brokers: []string{fmt.Sprintf("%s:%s", kafkaConfig.URL, kafkaConfig.Port)},
		Topic:   kafkaConfig.Topic,
		GroupID: kafkaConfig.GroupID,
	}
	redisClient := web.MakeRedisClient()
	service := NewGridService(redisClient)
	c := web.WithKafkaConsumer(r, consumer(service))
	instance := web.MakeServer(c, web.WithRedis(redisClient), web.WithGinEngine(func(r *gin.Engine) {}))
	instance.Run()
}
