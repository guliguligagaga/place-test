package web

import (
	"backend/logging"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
)

const (
	defaultServerAddr     = "0.0.0.0:8080"
	defaultShutdownTimer  = 3 * time.Second
	defaultMaxHeaderBytes = 1 << 20 // 1 MB
	defaultReadTimeout    = 10 * time.Second
	defaultWriteTimeout   = 10 * time.Second
	defaultIdleTimeout    = 30 * time.Second
)

func NewDefaultConfig() ServerConfig {
	return ServerConfig{
		Address:        defaultServerAddr,
		ShutdownTimer:  defaultShutdownTimer,
		MaxHeaderBytes: defaultMaxHeaderBytes,
		ReadTimeout:    defaultReadTimeout,
		WriteTimeout:   defaultWriteTimeout,
		IdleTimeout:    defaultIdleTimeout,
	}
}

type Server struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	router         *gin.Engine
	redis          redis.UniversalClient
	kafkaConsumers []*kafka.Reader
	kafkaWriters   []*kafka.Writer
	shutdownHooks  []io.Closer
	startupHooks   []func(ctx context.Context)
	healthChecks   []func(ctx context.Context) error
	config         ServerConfig
}

type ServerConfig struct {
	Address        string
	ShutdownTimer  time.Duration
	MaxHeaderBytes int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

type ServerOption func(*Server)

func NewServer(opts ...ServerOption) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		ctx:           ctx,
		cancelFunc:    cancel,
		shutdownHooks: make([]io.Closer, 0),
		startupHooks:  make([]func(context.Context), 0),
		healthChecks:  make([]func(context.Context) error, 0),
		config:        NewDefaultConfig(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithContext(ctx context.Context) ServerOption {
	return func(s *Server) {
		if s.cancelFunc != nil {
			s.cancelFunc()
		}
		s.ctx, s.cancelFunc = context.WithCancel(ctx)
	}
}

func WithGinEngine(routerConfig func(r *gin.Engine)) ServerOption {
	return func(s *Server) {
		s.router = gin.Default()
		s.router.Use(
			logging.Ginrus(),
			recoveryMiddleware(),
		)
		routerConfig(s.router)
	}
}

func WithKafkaConsumer(cfg kafka.ReaderConfig, handler func(k *kafka.Reader)) ServerOption {
	return func(s *Server) {
		reader := kafka.NewReader(cfg)
		s.kafkaConsumers = append(s.kafkaConsumers, reader)
		s.startupHooks = append(s.startupHooks, func(ctx context.Context) {
			handler(reader)
		})
		s.shutdownHooks = append(s.shutdownHooks, reader)
	}
}

func WithRedis(client redis.UniversalClient) ServerOption {
	return func(s *Server) {
		s.redis = client
		s.healthChecks = append(s.healthChecks, func(ctx context.Context) error {
			return client.Ping(ctx).Err()
		})
		s.shutdownHooks = append(s.shutdownHooks, client)
	}
}

func DefaultRedis() redis.UniversalClient {
	return newDefaultRedisClient()
}

func WithDefaultRedis() ServerOption {
	return func(s *Server) {
		WithRedis(newDefaultRedisClient())(s)
	}
}

func WithBackgroundWorker(worker func(ctx context.Context)) ServerOption {
	return func(s *Server) {
		s.startupHooks = append(s.startupHooks, func(ctx context.Context) {
			go worker(ctx)
		})
	}
}

func (s *Server) RegisterShutdownHook(closer io.Closer) {
	s.shutdownHooks = append(s.shutdownHooks, closer)
}

func (s *Server) RegisterHealthCheck(check func(ctx context.Context) error) {
	s.healthChecks = append(s.healthChecks, check)
}

func (s *Server) Redis() redis.UniversalClient {
	return s.redis
}

func (s *Server) Run() {
	defer s.cancelFunc() // Ensure context is cancelled when we exit

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for _, hook := range s.startupHooks {
		hook(s.ctx)
	}

	if s.router != nil {
		s.registerHealthEndpoints()
		go s.startHTTPServer()
	}

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.config.ShutdownTimer)
	defer shutdownCancel()

	s.cancelFunc()

	for _, hook := range s.shutdownHooks {
		if err := hook.Close(); err != nil {
			logging.Errorf("Shutdown hook error: %v", err)
		}
	}

	select {
	case <-shutdownCtx.Done():
		logging.Errorf("Shutdown timed out")
	default:
		logging.Infof("Shutdown completed successfully")
	}
}

func (s *Server) startHTTPServer() {
	srv := &http.Server{
		Addr:           s.config.Address,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	s.shutdownHooks = append(s.shutdownHooks, srv)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logging.Errorf("HTTP server error: %v", err)
		s.cancelFunc()
	}
}
func (s *Server) startBackgroundWorkers() {
	for _, hook := range s.startupHooks {
		go hook(s.ctx)
	}
}

func (s *Server) handleGracefulShutdown(quit <-chan os.Signal) {
	<-quit
	logging.Errorf("Initiating shutdown sequence...")

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimer)
	defer cancel()

	for _, hook := range s.shutdownHooks {
		if err := hook.Close(); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	select {
	case <-ctx.Done():
		logging.Errorf("Shutdown timed out")
	default:
		logging.Infof("Shutdown completed successfully")
	}
}

func (s *Server) registerHealthEndpoints() {
	s.router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})

	s.router.GET("/readyz", func(c *gin.Context) {
		if s.isReady() {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
	})
}

func (s *Server) isReady() bool {
	for _, check := range s.healthChecks {
		if err := check(s.ctx); err != nil {
			log.Printf("Health check failed: %v", err)
			return false
		}
	}
	return true
}

func newDefaultRedisClient() *redis.Client {
	addr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				logging.Errorf("Panic recovered: %v\nStack trace: %s", err, string(stack))

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}
