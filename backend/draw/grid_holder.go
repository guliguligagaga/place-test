package draw

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

type CellBroadcast struct {
	writer *kafka.Writer
}

func NewGridHolder(writer *kafka.Writer) *CellBroadcast {
	holder := &CellBroadcast{
		writer: writer,
	}
	return holder
}

type msg struct {
	Time int64  `json:"time,omitempty"`
	Data string `json:"data,omitempty"`
}

func (gh *CellBroadcast) updateCell(req *Req) error {
	message, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal update message: %v", err)
		return err
	}

	m := msg{
		Time: time.Now().UnixMilli(),
		Data: string(message),
	}

	m2, err := json.Marshal(&m)
	if err != nil {
		log.Printf("Failed to marshal update message: %v", err)
		return err
	}
	err = gh.writer.WriteMessages(context.Background(), kafka.Message{
		Value: m2,
	})
	return err
}
