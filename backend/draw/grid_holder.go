package draw

import (
	"backend/internal/protocol"
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type CellBroadcast struct {
	stream string
	writer redis.UniversalClient
}

func NewGridHolder(stream string, writer redis.UniversalClient) *CellBroadcast {
	holder := &CellBroadcast{
		stream: stream,
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
	_, err := gh.writer.XAdd(context.Background(), &redis.XAddArgs{
		Stream: gh.stream,
		Values: map[string]interface{}{
			"values": string(bytes[:]), // Store encoded cell as "values" field
		},
	}).Result()
	return err
}
