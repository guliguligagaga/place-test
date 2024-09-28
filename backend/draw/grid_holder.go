package draw

import (
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

type CellBroadcast struct {
	broadcastChan chan Req
	writer        *kafka.Writer
}

func NewGridHolder(writer *kafka.Writer) *CellBroadcast {
	holder := &CellBroadcast{
		broadcastChan: make(chan Req),
		writer:        writer,
	}
	return holder
}

func (gh *CellBroadcast) updateCell(req *Req) error {
	message, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal update message: %v", err)
		return err
	}
	err = gh.writer.WriteMessages(ctx, kafka.Message{
		Value: message,
	})
	return err
}
