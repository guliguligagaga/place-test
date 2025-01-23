package draw

import (
	"backend/web"
	"github.com/gin-gonic/gin"
)

type Req struct {
	X     uint16 `json:"x"`
	Y     uint16 `json:"y"`
	Color uint8  `json:"color"`
}

func Run() {
	redisClient := web.MakeRedisClient()
	gridHolder := NewGridHolder("grid_updates", redisClient)
	ginEngine := web.WithGinEngine(func(r *gin.Engine) {
		r.POST("/api/draw", func(c *gin.Context) {
			modifyCell(c, gridHolder)
		})
	})
	instance := web.MakeServer(ginEngine, web.WithRedis(redisClient))
	instance.Run()
}
