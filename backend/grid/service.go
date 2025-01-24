package grid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"backend/internal/protocol"
	"backend/logging"
	"github.com/go-redis/redis/v8"
)

const (
	StreamName         = "grid_updates"
	BroadcastChannel   = "grid_updates_brd"
	ConsumerGroup      = "grid-sync-consumer-group"
	LatestEpochKey     = "latest_epoch"
	UpdatesKeyPrefix   = "updates"
	KeyEnvVar          = "REDIS_GRID_KEY"
	PodNameEnvVar      = "POD_NAME"
	MaxRetries         = 3
	BaseRetryDelay     = 100 * time.Millisecond
	ProcessingTimeout  = 5 * time.Second
	MessageIDTTL       = 24 * time.Hour
	BatchSize          = 50
	Size               = 100
	MaxProcessingConns = 10
)

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
	ErrMessageTooShort      = errors.New("message too short")
)

type RedisClient interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
	XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) *redis.StatusCmd
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	BitField(ctx context.Context, key string, args ...interface{}) *redis.IntSliceCmd
}

type Service struct {
	redisClient RedisClient
	config      Config
	ctx         context.Context
	msgChan     chan redis.XMessage
}

type Config struct {
	GridKey   string
	PodName   string
	GridSize  uint16
	BatchSize int
}

func NewGridService(rc RedisClient) *Service {
	config := Config{
		GridKey:   os.Getenv(KeyEnvVar),
		PodName:   os.Getenv(PodNameEnvVar),
		GridSize:  Size,
		BatchSize: BatchSize,
	}

	service := &Service{
		ctx:         context.Background(),
		redisClient: rc,
		config:      config,
		msgChan:     make(chan redis.XMessage, 1000),
	}

	service.ensureConsumerGroup()

	return service
}

func (s *Service) ensureConsumerGroup() {
	err := s.redisClient.XGroupCreateMkStream(s.ctx, StreamName, ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		logging.Fatalf("failed to create consumer group %v", err)
	}
}

func (s *Service) Start(ctx context.Context) {
	logging.Infof("starting grid service")

	s.ctx = ctx

	for range MaxProcessingConns {
		go s.processMessages()
	}

	go s.consumeStream()

	<-ctx.Done()
	logging.Infof("shutting down grid service")
}

func (s *Service) consumeStream() {
	for {
		select {
		case <-s.ctx.Done():
			close(s.msgChan)

			return
		default:
			s.readStreamBatch()
		}
	}
}

func (s *Service) readStreamBatch() {
	ctx, cancel := context.WithTimeout(s.ctx, ProcessingTimeout)
	defer cancel()

	streams, err := s.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: s.config.PodName,
		Streams:  []string{StreamName, ">"},
		Count:    int64(s.config.BatchSize),
		Block:    time.Second,
	}).Result()

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return
	}

	if err != nil && !errors.Is(err, redis.Nil) {
		logging.Errorf("stream read error %v", err)
		time.Sleep(BaseRetryDelay)

		return
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			select {
			case s.msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *Service) processMessages() {
	for msg := range s.msgChan {
		if err := s.processMessageWithRetry(msg); err != nil {
			logging.Errorf("failed to process message %s after retries: %v", msg.ID, err)
		}
	}
}

func (s *Service) processMessageWithRetry(msg redis.XMessage) error {
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		err := s.processMessage(msg)
		if err == nil {
			return nil
		}

		if attempt < MaxRetries {
			logging.Warnf("retrying  %d message %s processing. err: %v", attempt, msg.ID, err)
			time.Sleep(BaseRetryDelay * time.Duration(attempt))
		}
	}

	return fmt.Errorf("max retries exceeded for message %s", msg.ID)
}

func (s *Service) processMessage(msg redis.XMessage) error {
	processedKey := "processed:" + msg.ID
	exists, err := s.redisClient.Exists(s.ctx, processedKey).Result()

	if err != nil {
		return fmt.Errorf("deduplication check failed: %w", err)
	}

	if exists > 0 {
		logging.Debugf("skipping duplicate message %s", msg.ID)

		return s.ackMessage(msg.ID)
	}

	if err = s.handleMessage(msg); err != nil {
		return fmt.Errorf("message handling failed: %w", err)
	}

	if err = s.redisClient.SetNX(s.ctx, processedKey, 1, MessageIDTTL).Err(); err != nil {
		return fmt.Errorf("failed to mark message as processed: %w", err)
	}

	return s.ackMessage(msg.ID)
}

func (s *Service) handleMessage(msg redis.XMessage) error {
	messageValue, ok := msg.Values["values"].(string)
	if !ok {
		return fmt.Errorf("%w: missing values field", ErrInvalidMessageFormat)
	}

	if len(messageValue) < 8 {
		return fmt.Errorf("%w: expected at least 8 bytes", ErrMessageTooShort)
	}

	timestamp, err := parseMessageTimestamp(msg.ID)
	if err != nil {
		return fmt.Errorf("timestamp parsing failed: %w", err)
	}

	updateTime := time.Now().UnixMilli()
	updateEpoch := updateTime / 60_000

	if err = s.storeUpdate(updateEpoch, messageValue, timestamp); err != nil {
		return fmt.Errorf("store update failed: %w", err)
	}

	if err = s.updateLatestEpoch(updateEpoch); err != nil {
		return fmt.Errorf("epoch update failed: %w", err)
	}

	if err = s.updateGridStatus(messageValue); err != nil {
		return fmt.Errorf("grid status update failed: %w", err)
	}

	if err = s.broadcastUpdate(messageValue); err != nil {
		return fmt.Errorf("broadcast failed: %w", err)
	}

	return nil
}

func parseMessageTimestamp(id string) (int64, error) {
	timestampStr := strings.Split(id, "-")[0]

	return strconv.ParseInt(timestampStr, 10, 64)
}

func (s *Service) storeUpdate(epoch int64, value string, timestamp int64) error {
	key := fmt.Sprintf("%s:%s:%d", s.config.GridKey, UpdatesKeyPrefix, epoch)

	return s.redisClient.ZAdd(s.ctx, key, &redis.Z{
		Score:  float64(timestamp),
		Member: value,
	}).Err()
}

func (s *Service) updateLatestEpoch(epoch int64) error {
	return s.redisClient.Set(s.ctx, LatestEpochKey, epoch, 0).Err()
}

func (s *Service) updateGridStatus(value string) error {
	cell := protocol.Decode([8]byte([]byte(value)))
	offset := calculateOffset(cell.Y, cell.X, s.config.GridSize)
	_, err := s.redisClient.BitField(s.ctx, s.config.GridKey, "SET", "u4", offset, cell.Color).Result()

	return err
}

func calculateOffset(y, x, gridSize uint16) int64 {
	byteIndex := (y*gridSize + x) / 2
	isUpperNibble := (x & 1) == 0

	if isUpperNibble {
		return int64(byteIndex * 8)
	}
	return int64(byteIndex*8 + 4)
}

func (s *Service) broadcastUpdate(value string) error {
	return s.redisClient.Publish(s.ctx, BroadcastChannel, value).Err()
}

func (s *Service) ackMessage(msgID string) error {
	return s.redisClient.XAck(s.ctx, StreamName, ConsumerGroup, msgID).Err()
}
