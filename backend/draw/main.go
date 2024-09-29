package draw

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"os"
	"time"
	"web"
)

type Req struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Color int `json:"color"`
}

var ctx = context.Background()

func Run() {
	kafkaWriter := makeWriter()
	gridHolder := NewGridHolder(kafkaWriter)
	ginEngine := web.WithGinEngine(func(r *gin.Engine) {
		r.POST("/api/draw", func(c *gin.Context) {
			modifyCell(c, gridHolder)
		})
	})
	instance := web.MakeServer(ginEngine, web.WithKafkaWriter(kafkaWriter))
	instance.Run()
}

func makeWriter() *kafka.Writer {
	kafkaUrl := fmt.Sprintf("%s:%s", os.Getenv("KAFKA_URL"), os.Getenv("KAFKA_PORT"))
	return &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv(kafkaUrl)),
		Topic:                  "grid_updates",
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		BatchSize:              1_000,
		Async:                  true,
		BatchTimeout:           50 * time.Millisecond,
		RequiredAcks:           kafka.RequireOne,
		Compression:            kafka.Snappy,
	}
}
