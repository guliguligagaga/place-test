package main

import (
	"time"
)

const (
	updatesTopic     = "grid-updates"
	kafkaGroupID     = "grid-sync-consumer-group"
	redisKeyPrefix   = "grid:update:"
	redisCacheWindow = time.Hour * 24 // Cache updates for 24 hours
)

type Update struct {
	Version int64     `json:"version"`
	X       int       `json:"x"`
	Y       int       `json:"y"`
	Color   string    `json:"color"`
	Time    time.Time `json:"time"`
}

func main() {

}
