package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var req struct {
	Provider string `json:"provider" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

func signIn(c *gin.Context) {
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	provider, err := GetProvider(req.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown provider"})
		return
	}

	tokenInfo, err := provider.SignIn(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	jwtToken, err := generateJWT(tokenInfo, provider.Name())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
	})
}

func access(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		tokenString = c.Query("token")
	} else {
		tokenString = tokenString[len("Bearer "):]
	}

	if tokenString == "" {
		log.Println("Missing authorization header")
		c.Status(http.StatusUnauthorized)
		return
	}

	_, err := validateJWTToken(tokenString)
	if err != nil {
		log.Printf("Invalid token: %v", err)
		c.Status(http.StatusUnauthorized)
		return
	}
	c.Status(http.StatusOK)
}
