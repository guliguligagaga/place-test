package main

import (
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine) {
	gr := router.Group("/api/auth")
	gr.POST("/google", googleSignIn)
	gr.GET("/verify", verifyToken)
	gr.GET("/validate_token", validateToken)
}
