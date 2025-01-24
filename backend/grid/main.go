package grid

import (
	"context"

	"backend/web"
	"github.com/gin-gonic/gin"
)

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redis := web.DefaultRedis()
	s := NewGridService(redis)

	server := web.NewServer(
		web.WithContext(ctx),
		web.WithRedis(redis),
		web.WithGinEngine(func(r *gin.Engine) {}),
		web.WithBackgroundWorker(s.Start),
	)

	server.Run()
}
