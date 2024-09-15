package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	ss "strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/oauth2/v2"
)

func verifyGoogleToken(token string) (*GoogleTokenInfo, error) {
	ctx := context.Background()
	oauth2Service, err := oauth2.NewService(ctx)
	if err != nil {
		return nil, err
	}

	tokenInfo, err := oauth2Service.Tokeninfo().IdToken(token).Do()
	if err != nil {
		return nil, err
	}

	return &GoogleTokenInfo{
		Sub:   tokenInfo.UserId,
		Email: tokenInfo.Email,
	}, nil
}

type req struct {
	Token string `json:"token,omitempty"`
}

func googleSignIn(c *gin.Context) {
	var r req
	if err := json.NewDecoder(c.Request.Body).Decode(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tokenInfo, err := verifyGoogleToken(r.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	claims := &Claims{
		Sub: tokenInfo.Sub,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": signedToken,
		"user": gin.H{
			"email": tokenInfo.Email,
		},
	})
}

func verifyToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	tokenString = tokenString[len("Bearer "):]

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	if _, ok := token.Claims.(*Claims); ok && token.Valid {
		c.JSON(http.StatusOK, gin.H{"message": "Valid token"})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
	}
}

func validateToken(c *gin.Context) {
	var tokenString string

	// Check for token in Sec-WebSocket-Protocol header
	if wsProtocol := c.GetHeader("Sec-WebSocket-Protocol"); wsProtocol != "" {
		parts := ss.SplitN(wsProtocol, ".", 2)
		if len(parts) == 2 && parts[0] == "token" {
			tokenString = parts[1]
		}
		c.Header("Sec-WebSocket-Protocol", "") // Remove token from header
	}

	// If not found in Sec-WebSocket-Protocol, check URL query parameter
	if tokenString == "" {
		tokenString = c.Query("token")
	}

	// If still not found, check X-Auth-Token header (for non-WebSocket requests)
	if tokenString == "" {
		tokenString = c.GetHeader("X-Auth-Token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	if _, ok := token.Claims.(*Claims); ok && token.Valid {
		// Token is valid, you can use claims if needed
		c.JSON(http.StatusOK, gin.H{"message": "Valid token"})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
	}
}
