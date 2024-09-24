package main

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
	"runtime"
	"sync"
	"time"
)

func main() {
	// Kafka writer configuration
	writer := kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "grid_updates",
	}
	defer writer.Close()

	// Generate load
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			writeMessages(&writer)
		}()
	}
	wg.Wait()
}

func writeMessages(k *kafka.Writer) {
	for i := 0; i < 1000; i++ {
		err := k.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(time.Now().String()),
				Value: []byte("Test message"),
			},
		)
		if err != nil {
			log.Println("Failed to write message:", err)
		} else {
			log.Printf("Message written: %d\n", i)
		}
	}
}
