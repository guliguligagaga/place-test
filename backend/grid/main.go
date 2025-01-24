package grid

import (
	"backend/web"
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	redis := web.DefaultRedis()
	s := NewGridService(redis, logger)

	server := web.NewServer(
		web.WithContext(ctx),
		web.WithRedis(redis),
		web.WithGinEngine(func(r *gin.Engine) {}),
		web.WithBackgroundWorker(s.Start),
	)

	server.Run()
}
