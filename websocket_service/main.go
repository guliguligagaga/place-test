package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

type Clients struct {
	sync.RWMutex
	clients map[string]*websocket.Conn
}

var clients = Clients{
	clients: make(map[string]*websocket.Conn),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("%d", len(clients.clients)+1)
	clients.Lock()
	clients.clients[clientID] = conn
	clients.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received message from client %s: %s", clientID, msg)
	}

	clients.Lock()
	delete(clients.clients, clientID)
	clients.Unlock()
}

func kafkaConsumer(ctx context.Context, brokers []string, topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "websocket_service",
	})

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Println("Kafka read error:", err)
			return
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		clients.RLock()
		for _, conn := range clients.clients {
			err := conn.WriteMessage(websocket.TextMessage, m.Value)
			if err != nil {
				log.Println("Write error:", err)
			}
		}
		clients.RUnlock()
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	server := &http.Server{Addr: ":8081"}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go kafkaConsumer(ctx, []string{"localhost:9092"}, "grid_updates")

	<-ctx.Done()
	log.Println("Shutdown signal received")

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}
