package draw

import (
	"backend/web"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"os"
	"time"
)

type Req struct {
	X     uint16 `json:"x"`
	Y     uint16 `json:"y"`
	Color uint8  `json:"color"`
}

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
		Addr:                   kafka.TCP(kafkaUrl),
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
