package grid

import (
	"backend/web"
	"context"
	"github.com/gin-gonic/gin"
)

const (
	size             = 100
	UpdatesKeyPrefix = "updates"
	LatestEpochKey   = "latest_epoch"
)

type Update struct {
	Timestamp uint64 `json:"time"`
	Data      string `json:"data"`
}

func Run() {
	redisClient := web.MakeRedisClient()
	ctx, cancelFunc := context.WithCancel(context.Background())
	s := NewGridService(ctx, redisClient)
	instance := web.MakeServer(web.WithConsumer(s.consumer), web.WithRedis(redisClient), web.WithGinEngine(func(r *gin.Engine) {}))
	instance.AddOnExit(cancelFunc)
	instance.Run()
}
