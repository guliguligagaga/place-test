package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/idtoken"
)

var googleClientId = os.Getenv("GOOGLE_CLIENT_ID")

func verifyGoogleToken(token string) (*GoogleTokenInfo, error) {
	log.Println("Verifying Google token...")
	ctx := context.Background()

	payload, err := idtoken.Validate(ctx, token, googleClientId)
	if err != nil {
		log.Printf("Error validating Google token: %v", err)
		return nil, err
	}

	log.Printf("Google token verified successfully. Subject: %s, Email: %s", payload.Subject, payload.Claims["email"])
	return &GoogleTokenInfo{
		Sub:   payload.Subject,
		Email: payload.Claims["email"].(string),
	}, nil
}

type req struct {
	Token string `json:"token,omitempty"`
}

func googleSignIn(c *gin.Context) {
	log.Println("Handling Google Sign In request...")
	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tokenInfo, err := verifyGoogleToken(r.Token)
	if err != nil {
		log.Printf("Invalid Google token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	log.Printf("Google token verified. Generating JWT for user: %s", tokenInfo.Email)
	claims := &Claims{
		Sub: tokenInfo.Sub,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	log.Println("JWT generated successfully")
	c.JSON(http.StatusOK, gin.H{
		"token": signedToken,
		"user": gin.H{
			"email": tokenInfo.Email,
		},
	})
}

func verifyToken(c *gin.Context) {
	log.Println("Verifying token from Authorization header...")
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		log.Println("Missing authorization header")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	tokenString = tokenString[len("Bearer "):]

	claims, err := validateJWTToken(tokenString)
	if err != nil {
		log.Printf("Invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	log.Printf("Token verified successfully. Subject: %s", claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "Valid token", "sub": claims.Sub})
}

func validateToken(c *gin.Context) {
	log.Println("Validating token from query or header...")
	tokenString := c.Query("token")
	if tokenString == "" {
		log.Println("missing query token")
		tokenString = c.GetHeader("X-Auth-Token")
	}

	if tokenString == "" {
		log.Printf("headers %v", c.Request.Header)
		log.Println("Missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	claims, err := validateJWTToken(tokenString)
	if err != nil {
		log.Printf("Invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	log.Printf("Token validated successfully. Subject: %s", claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "Valid token", "sub": claims.Sub})
}

func validateJWTToken(tokenString string) (*Claims, error) {
	log.Println("Validating JWT token...")
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("Unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		log.Printf("Error parsing JWT token: %v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		log.Println("JWT token validated successfully")
		return claims, nil
	}

	log.Println("Invalid JWT token")
	return nil, errors.New("invalid token")
}
