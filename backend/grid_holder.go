package main

import (
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

type CellBroadcast struct {
	broadcastChan chan DrawReq
	writer        *kafka.Writer
}

func NewGridHolder(writer *kafka.Writer) *CellBroadcast {
	holder := &CellBroadcast{
		broadcastChan: make(chan DrawReq),
		writer:        writer,
	}
	return holder
}

func (gh *CellBroadcast) updateCell(req *DrawReq) error {
	message, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal update message: %v", err)
		return err
	}
	err = gh.writer.WriteMessages(ctx, kafka.Message{
		Value: message,
	})
	if err != nil {
		log.Printf("Failed to send update to Kafka: %v", err)
	}
	return nil
}
