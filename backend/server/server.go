package server

import (
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Instance struct {
	closeOnExit []io.Closer
	onStart     []func() error
	route       *gin.Engine

	pingFunctions []func() error
}

func NewInstance() *Instance {
	return &Instance{}
}

func (i *Instance) AddCloseOnExit(c io.Closer) {
	i.closeOnExit = append(i.closeOnExit, c)
}

func (i *Instance) AddOnStart(f func() error) {
	i.onStart = append(i.onStart, f)
}

func (i *Instance) SetRoute(r *gin.Engine) {
	i.route = r
}

func (i *Instance) AddPingFunction(f func() error) {
	i.pingFunctions = append(i.pingFunctions, f)
}

func (i *Instance) Run() {
	quit := make(chan os.Signal, 1)
	if i.route != nil {
		i.registerHealthRoutes(i.route)
		go func() {
			if err := http.ListenAndServe("0.0.0.0:8080", i.route); err != nil {
				log.Printf("Error starting server: %v\n", err)
				quit <- syscall.SIGTERM
			}
		}()

	}

	for _, f := range i.onStart {
		go func() {
			if err := f(); err != nil {
				log.Printf("Error starting: %v", err)
				quit <- syscall.SIGTERM
			}
		}()
	}

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	for _, cl := range i.closeOnExit {
		if err := cl.Close(); err != nil {
			log.Printf("Error closing: %v", err)
		}
	}
}

func (i *Instance) registerHealthRoutes(route *gin.Engine) {
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

func (i *Instance) checkReadiness() bool {
	for _, f := range i.pingFunctions {
		if err := f(); err != nil {
			log.Printf("Error pinging: %v", err)
			return false
		}
	}
	return true
}
