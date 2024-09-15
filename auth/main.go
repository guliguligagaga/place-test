package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"

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
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}

	router.Use(cors.New(config))
	registerRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // default port
	}
	log.Fatal(router.Run("0.0.0.0:" + port))
}
