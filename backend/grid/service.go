package grid

import (
	"backend/internal/protocol"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"os"
	"strconv"
	"time"
)

type Service struct {
	redisClient *redis.Client
	gridKey     string
}

func NewGridService(redisClient *redis.Client) *Service {
	return &Service{
		redisClient: redisClient,
		gridKey:     os.Getenv("REDIS_GRID_KEY"),
	}
}

func consumer(s *Service) func(k *kafka.Reader) {
	return func(k *kafka.Reader) {
		for {
			m, err := k.ReadMessage(context.Background())
			if err != nil {
				fmt.Println("failed to read message", err.Error())
			}
			if err = s.handle(context.Background(), m); err != nil {
				fmt.Printf("failed to handle message: %s", err.Error())
			}
		}
	}
}
func (s *Service) handle(ctx context.Context, msg kafka.Message) error {
	updateTime := time.Now().UnixMilli()
	updateEpoch := updateTime / 60_000

	if err := s.storeUpdate(ctx, updateEpoch, msg); err != nil {
		return fmt.Errorf("failed to store update: %w", err)
	}

	if err := s.updateLatestEpoch(ctx, updateEpoch); err != nil {
		return fmt.Errorf("failed to update epoch: %w", err)
	}

	if err := s.updateGridStatus(ctx, msg.Value); err != nil {
		return fmt.Errorf("failed to update grid status: %w", err)
	}

	return nil
}

func (s *Service) storeUpdate(ctx context.Context, epoch int64, msg kafka.Message) error {
	return s.redisClient.ZAdd(ctx,
		fmt.Sprintf("%s%d", UpdatesKeyPrefix, epoch),
		&redis.Z{
			Score:  float64(msg.Time.UnixMilli()),
			Member: string(msg.Value),
		}).Err()
}

func (s *Service) updateLatestEpoch(ctx context.Context, epoch int64) error {
	return s.redisClient.Set(ctx,
		LatestEpochKey,
		strconv.FormatInt(epoch, 10),
		0).Err()
}

func (s *Service) updateGridStatus(ctx context.Context, value []byte) error {
	fmt.Printf("Updating grid status with value: %b\n", value)
	cell := protocol.Decode([8]byte(value))
	fmt.Printf("Decoded cell: %v\n", cell)
	offset := calculateOffset(cell.Y, cell.X)
	fmt.Printf("Calculated offset: %d\n", offset)
	_, err := s.redisClient.BitField(ctx,
		s.gridKey,
		"SET",
		"u4",
		offset,
		cell.Color).Result()
	return err
}

func calculateOffset(y, x uint16) int64 {
	byteIndex := (y*GridSize + x) / 2
	isUpperNibble := (x & 1) == 0

	if isUpperNibble {
		return int64(byteIndex * 8)
	}
	return int64(byteIndex*8 + 4)
}
