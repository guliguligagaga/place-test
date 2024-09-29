package auth

import (
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine) {
	gr := router.Group("/api/auth")
	gr.POST("/signIn", signIn)
	gr.GET("/access", access)
	gr.POST("/renew", renewToken)
}
