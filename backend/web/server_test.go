package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

type MockCloser struct {
	closeWasCalled bool
	shouldError    bool
}

func (m *MockCloser) Close() error {
	m.closeWasCalled = true
	if m.shouldError {
		return fmt.Errorf("mock close error")
	}
	return nil
}

type MockRedis struct {
	*redis.Client
	MockCloser
	pingError error
}

func (m *MockRedis) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetErr(m.pingError)
	return cmd
}

func (m *MockRedis) Context() context.Context {
	return context.Background()
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with no options", func(t *testing.T) {
		s := NewServer()
		assert.NotNil(t, s)
	})

	t.Run("creates server with gin engine", func(t *testing.T) {
		s := NewServer(WithGinEngine(func(r *gin.Engine) {
			r.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})
		}))
		assert.NotNil(t, s.router)

		// Test the added route
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		s.router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})

	t.Run("creates server with redis", func(t *testing.T) {
		mockRedis := &redis.Client{}
		s := NewServer(WithRedis(mockRedis))
		assert.Equal(t, mockRedis, s.redis)
		assert.Len(t, s.healthChecks, 1)
		assert.Contains(t, s.shutdownHooks, mockRedis)
	})

	t.Run("creates server with kafka consumer", func(t *testing.T) {
		cfg := kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "test-topic",
			GroupID: "test-group",
		}

		var readerCalled bool

		s := NewServer(WithKafkaConsumer(cfg, func(k *kafka.Reader) {
			readerCalled = true
		}))

		assert.Len(t, s.kafkaConsumers, 1)
		assert.Len(t, s.startupHooks, 1)
		assert.Len(t, s.shutdownHooks, 1)

		// Test that the callback is called
		s.startupHooks[0](context.Background())
		assert.True(t, readerCalled)
	})
}

func TestHealthEndpoints(t *testing.T) {
	t.Run("healthz returns 200", func(t *testing.T) {
		s := NewServer(WithGinEngine(func(r *gin.Engine) {}))
		s.registerHealthEndpoints()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/healthz", nil)
		s.router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "alive", response["status"])
	})

	t.Run("readyz returns 200 when all ping functions succeed", func(t *testing.T) {
		s := NewServer(WithGinEngine(func(r *gin.Engine) {}))
		s.healthChecks = append(s.healthChecks, func(ctx context.Context) error { return nil })
		s.registerHealthEndpoints()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/readyz", nil)
		s.router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ready", response["status"])
	})

	t.Run("readyz returns 503 when ping function fails", func(t *testing.T) {
		s := NewServer(WithGinEngine(func(r *gin.Engine) {}))
		s.healthChecks = append(s.healthChecks, func(ctx context.Context) error { return fmt.Errorf("ping failed") })
		s.registerHealthEndpoints()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/readyz", nil)
		s.router.ServeHTTP(w, req)

		assert.Equal(t, 503, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not ready", response["status"])
	})
}

func TestServerShutdown(t *testing.T) {
	t.Run("closes all resources on shutdown", func(t *testing.T) {
		mock1 := &MockCloser{}
		mock2 := &MockCloser{}

		s := NewServer()
		s.RegisterShutdownHook(mock1)
		s.RegisterShutdownHook(mock2)

		// Start the server in a goroutine
		go func() {
			s.Run()
		}()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Trigger shutdown
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)

		// Give it a moment to shut down
		time.Sleep(100 * time.Millisecond)

		assert.True(t, mock1.closeWasCalled)
		assert.True(t, mock2.closeWasCalled)
	})

	t.Run("handles close errors gracefully", func(t *testing.T) {
		mock := &MockCloser{shouldError: true}

		s := NewServer()
		s.RegisterShutdownHook(mock)

		go func() {
			s.Run()
		}()

		time.Sleep(100 * time.Millisecond)

		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)

		assert.True(t, mock.closeWasCalled)
	})
}

func TestRedisIntegration(t *testing.T) {
	t.Run("redis ping function succeeds when redis is healthy", func(t *testing.T) {
		mockRedis := &MockRedis{}
		s := NewServer(WithRedis(mockRedis))

		assert.Len(t, s.healthChecks, 1)
		err := s.healthChecks[0](context.Background())
		assert.NoError(t, err)
	})

	t.Run("redis ping function fails when redis is unhealthy", func(t *testing.T) {
		mockRedis := &MockRedis{pingError: fmt.Errorf("redis connection failed")}
		s := NewServer(WithRedis(mockRedis))

		assert.Len(t, s.healthChecks, 1)
		err := s.healthChecks[0](context.Background())
		assert.Error(t, err)
	})
}
