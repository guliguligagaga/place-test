package main

import (
	"context"
	"net/http"
	"os"
	"server"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

type DrawReq struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Color int `json:"color"`
}

var ctx = context.Background()

func main() {

	kafkaWriter := &kafka.Writer{
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

	instance := server.NewInstance()
	instance.AddCloseOnExit(kafkaWriter)

	gridHolder := NewGridHolder(kafkaWriter)
	router := gin.Default()
	router.POST("/draw", func(c *gin.Context) {
		modifyCell(c, gridHolder)
	})

	address := getEnv("BIND_ADDRESS", "0.0.0.0:5001")
	s := &http.Server{
		Addr:    address,
		Handler: router,
	}

	instance.AddOnStart(s.ListenAndServe)
	instance.Run()
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
