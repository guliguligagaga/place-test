package draw

import (
	"backend/internal/protocol"
	"context"
	"github.com/segmentio/kafka-go"
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

func reqToCell(r *Req) protocol.Cell {
	return protocol.Cell{X: r.X, Y: r.Y, Color: r.Color, Time: time.Now().UnixMilli()}
}

func (gh *CellBroadcast) updateCell(req *Req) error {
	cell := reqToCell(req)
	bytes := cell.Encode()
	err := gh.writer.WriteMessages(context.Background(), kafka.Message{
		Value: bytes[:],
	})
	return err
}
