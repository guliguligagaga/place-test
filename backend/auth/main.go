package main

import (
	"os"
	"server"

	"github.com/gin-gonic/gin"
)

var jwtSecret []byte

func init() {
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("secret")
		//log.Fatal("JWT_SECRET environment variable is not set")
	}
}

func main() {
	router := gin.Default()

	_ = router.SetTrustedProxies(nil)
	instance := server.NewInstance()
	registerRoutes(router)

	instance.SetRoute(router)
	instance.Run()
}
