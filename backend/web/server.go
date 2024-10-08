package web

import (
	"backend/logging"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	closeOnExit    []io.Closer
	onStart        []func()
	route          *gin.Engine
	redis          *redis.Client
	pingFunctions  []func() error
	kafkaConsumers []*kafka.Reader
	kafkaWriters   []*kafka.Writer
}

type Opts func(*Server)

var WithGinEngine = func(f func(r *gin.Engine)) func(s *Server) {
	return func(s *Server) {
		s.route = gin.Default()
		s.route.Use(logging.Ginrus())
		f(s.route)
	}
}

var WithKafkaConsumer = func(cfg kafka.ReaderConfig, f func(k *kafka.Reader)) func(s *Server) {
	return func(s *Server) {
		r := kafka.NewReader(cfg)
		s.kafkaConsumers = append(s.kafkaConsumers, r)
		s.onStart = append(s.onStart, func() {
			f(r)
		})
		s.closeOnExit = append(s.closeOnExit, r)
	}
}

var WithRedis = func(s *Server) {
	url := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	s.redis = redis.NewClient(&redis.Options{
		Addr: url,
	})
	s.AddPingFunction(func() error {
		return s.redis.Ping(s.redis.Context()).Err()
	})
	s.closeOnExit = append(s.closeOnExit, s.redis)
}

var WithKafkaWriter = func(cfg *kafka.Writer) func(s *Server) {
	return func(s *Server) {
		s.kafkaWriters = append(s.kafkaWriters, cfg)
		s.closeOnExit = append(s.closeOnExit, cfg)
	}
}

func MakeServer(opts ...Opts) *Server {
	s := &Server{}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (i *Server) AddCloseOnExit(c io.Closer) {
	i.closeOnExit = append(i.closeOnExit, c)
}

func (i *Server) AddPingFunction(f func() error) {
	i.pingFunctions = append(i.pingFunctions, f)
}

func (i *Server) Redis() *redis.Client {
	return i.redis
}

func (i *Server) Run() {
	quit := make(chan os.Signal, 1)
	if i.route != nil {
		i.registerHealthRoutes(i.route)
		go func() {
			srv := &http.Server{Addr: "0.0.0.0:8080", Handler: i.route}
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("Error starting server: %v\n", err)
				quit <- syscall.SIGTERM
				i.closeOnExit = append(i.closeOnExit, srv)
			}
		}()

	}

	for _, f := range i.onStart {
		go f()
	}

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Errorf("Shutting down...")
	for _, cl := range i.closeOnExit {
		if err := cl.Close(); err != nil {
			log.Printf("Error closing: %v", err)
		}
	}
}

func (i *Server) registerHealthRoutes(route *gin.Engine) {
	// Liveness probe route
	route.GET("/healthz", func(c *gin.Context) {
		// Return a 200 status to indicate the service is alive
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
	})

	// Readiness probe route
	route.GET("/readyz", func(c *gin.Context) {
		isReady := i.checkReadiness()
		if isReady {
			c.JSON(http.StatusOK, gin.H{
				"status": "ready",
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
			})
		}
	})
}

func (i *Server) checkReadiness() bool {
	for _, f := range i.pingFunctions {
		if err := f(); err != nil {
			log.Printf("Error pinging: %v", err)
			return false
		}
	}
	return true
}
