package grid

import (
	"backend/internal/protocol"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	client  *redis.Client
	gridKey string
	ctx     context.Context
}

func NewGridService(ctx context.Context, redisClient *redis.Client) *Service {
	err := redisClient.XGroupCreate(context.Background(), "grid_updates", "grid-sync-consumer-group", "0").Err()
	if !strings.Contains(fmt.Sprint(err), "BUSYGROUP") {
		panic(err)
	}
	return &Service{
		ctx:     ctx,
		client:  redisClient,
		gridKey: os.Getenv("REDIS_GRID_KEY"),
	}
}

func (s *Service) consumer() error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		streams, err := s.client.XReadGroup(s.ctx, &redis.XReadGroupArgs{
			Group:    "grid-sync-consumer-group",
			Consumer: os.Getenv("POD_NAME"),
			Streams:  []string{"grid_updates", ">"},
			Count:    1,
			Block:    time.Second * 1,
		}).Result()

		if err != nil && err != redis.Nil {
			return err
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				err = s.handle(message)
				if err != nil {
					continue
				}

				s.client.XAck(s.ctx, "grid_updates", "grid-sync-consumer-group", message.ID)
			}
		}
	}
}

func (s *Service) handle(msg redis.XMessage) error {
	updateTime := time.Now().UnixMilli()
	updateEpoch := updateTime / 60_000
	var messageValue string

	if value, ok := msg.Values["values"].(string); ok {
		messageValue = value
	}

	timestampStr := strings.Split(msg.ID, "-")[0]
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse message timestamp: %w", err)
	}
	if err = s.storeUpdate(updateEpoch, messageValue, timestamp); err != nil {
		return fmt.Errorf("failed to store update: %w", err)
	}

	if err = s.updateLatestEpoch(updateEpoch); err != nil {
		return fmt.Errorf("failed to update epoch: %w", err)
	}

	if err = s.updateGridStatus([]byte(messageValue)); err != nil {
		return fmt.Errorf("failed to update grid status: %w", err)
	}

	return nil
}

func (s *Service) storeUpdate(epoch int64, messageValue string, timestamp int64) error {
	return s.client.ZAdd(s.ctx,
		fmt.Sprintf("%s:%s:%d", s.gridKey, UpdatesKeyPrefix, epoch),
		&redis.Z{
			Score:  float64(timestamp),
			Member: messageValue,
		}).Err()
}

func (s *Service) updateLatestEpoch(epoch int64) error {
	return s.client.Set(s.ctx,
		LatestEpochKey,
		strconv.FormatInt(epoch, 10),
		0).Err()
}

func (s *Service) updateGridStatus(value []byte) error {
	cell := protocol.Decode([8]byte(value))
	offset := calculateOffset(cell.Y, cell.X)
	_, err := s.client.BitField(s.ctx,
		s.gridKey,
		"SET",
		"u4",
		offset,
		cell.Color).Result()
	return err
}

func calculateOffset(y, x uint16) int64 {
	byteIndex := (y*size + x) / 2
	isUpperNibble := (x & 1) == 0

	if isUpperNibble {
		return int64(byteIndex * 8)
	}
	return int64(byteIndex*8 + 4)
}
