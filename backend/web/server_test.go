package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
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

func TestMakeServer(t *testing.T) {
	t.Run("creates server with no options", func(t *testing.T) {
		s := MakeServer()
		assert.NotNil(t, s)
		assert.Empty(t, s.closeOnExit)
		assert.Empty(t, s.onStart)
		assert.Nil(t, s.route)
		assert.Nil(t, s.redis)
	})

	t.Run("creates server with gin engine", func(t *testing.T) {
		s := MakeServer(WithGinEngine(func(r *gin.Engine) {
			r.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})
		}))
		assert.NotNil(t, s.route)

		// Test the added route
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		s.route.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})

	t.Run("creates server with redis", func(t *testing.T) {
		mockRedis := &redis.Client{}
		s := MakeServer(WithRedis(mockRedis))
		assert.Equal(t, mockRedis, s.redis)
		assert.Len(t, s.pingFunctions, 1)
		assert.Contains(t, s.closeOnExit, mockRedis)
	})

	t.Run("creates server with kafka consumer", func(t *testing.T) {
		cfg := kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "test-topic",
			GroupID: "test-group",
		}

		var readerCalled bool
		s := MakeServer(WithKafkaConsumer(cfg, func(k *kafka.Reader) {
			readerCalled = true
		}))

		assert.Len(t, s.kafkaConsumers, 1)
		assert.Len(t, s.onStart, 1)
		assert.Len(t, s.closeOnExit, 1)

		// Test that the callback is called
		s.onStart[0]()
		assert.True(t, readerCalled)
	})
}

func TestHealthEndpoints(t *testing.T) {
	t.Run("healthz returns 200", func(t *testing.T) {
		s := MakeServer(WithGinEngine(func(r *gin.Engine) {}))
		s.registerHealthRoutes(s.route)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/healthz", nil)
		s.route.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "alive", response["status"])
	})

	t.Run("readyz returns 200 when all ping functions succeed", func(t *testing.T) {
		s := MakeServer(WithGinEngine(func(r *gin.Engine) {}))
		s.AddPingFunction(func() error { return nil })
		s.registerHealthRoutes(s.route)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/readyz", nil)
		s.route.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ready", response["status"])
	})

	t.Run("readyz returns 503 when ping function fails", func(t *testing.T) {
		s := MakeServer(WithGinEngine(func(r *gin.Engine) {}))
		s.AddPingFunction(func() error { return fmt.Errorf("ping failed") })
		s.registerHealthRoutes(s.route)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/readyz", nil)
		s.route.ServeHTTP(w, req)

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

		s := MakeServer()
		s.AddCloseOnExit(mock1)
		s.AddCloseOnExit(mock2)

		// Start the server in a goroutine
		go func() {
			s.Run()
		}()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Trigger shutdown
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)

		// Give it a moment to shut down
		time.Sleep(100 * time.Millisecond)

		assert.True(t, mock1.closeWasCalled)
		assert.True(t, mock2.closeWasCalled)
	})

	t.Run("handles close errors gracefully", func(t *testing.T) {
		mock := &MockCloser{shouldError: true}

		s := MakeServer()
		s.AddCloseOnExit(mock)

		go func() {
			s.Run()
		}()

		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)

		assert.True(t, mock.closeWasCalled)
	})
}

func TestRedisIntegration(t *testing.T) {
	t.Run("redis ping function succeeds when redis is healthy", func(t *testing.T) {
		mockRedis := &MockRedis{}
		s := MakeServer(WithRedis(mockRedis))

		assert.Len(t, s.pingFunctions, 1)
		err := s.pingFunctions[0]()
		assert.NoError(t, err)
	})

	t.Run("redis ping function fails when redis is unhealthy", func(t *testing.T) {
		mockRedis := &MockRedis{pingError: fmt.Errorf("redis connection failed")}
		s := MakeServer(WithRedis(mockRedis))

		assert.Len(t, s.pingFunctions, 1)
		err := s.pingFunctions[0]()
		assert.Error(t, err)
	})
}
