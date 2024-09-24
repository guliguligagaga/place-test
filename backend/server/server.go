package server

import (
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Instance struct {
	closeOnExit []io.Closer
	onStart     []func() error
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

func (i *Instance) Run() {

	quit := make(chan os.Signal, 1)

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
