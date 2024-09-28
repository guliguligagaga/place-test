package draw

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
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
		r.POST("/draw", func(c *gin.Context) {
			modifyCell(c, gridHolder)
		})
	})
	instance := web.MakeServer(ginEngine, web.WithKafkaWriter(kafkaWriter))
	instance.Run()
}

func makeWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP("kafka:29092"),
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
