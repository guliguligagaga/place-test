package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type DrawReq struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Color int `json:"color"`
}

var ctx = context.Background()

func main() {

	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "grid_updates",
		Balancer: &kafka.LeastBytes{},
	}

	gridHolder := NewGridHolder(kafkaWriter)

	router := gin.Default()
	router.POST("/draw", func(c *gin.Context) {
		modifyCell(c, gridHolder)
	})

	address := getEnv("BIND_ADDRESS", "0.0.0.0:8080")
	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Could not listen on %s: %v\n", address, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
