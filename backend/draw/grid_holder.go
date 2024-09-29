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
	Req
	Time int64 `json:"time,omitempty"`
}

func (gh *CellBroadcast) updateCell(req *Req) error {
	m2, err := json.Marshal(&msg{
		Req:  *req,
		Time: time.Now().UnixMilli(),
	})
	if err != nil {
		log.Printf("Failed to marshal update message: %v", err)
		return err
	}
	err = gh.writer.WriteMessages(context.Background(), kafka.Message{
		Value: m2,
	})
	return err
}
