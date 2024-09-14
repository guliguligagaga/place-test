package main

import (
	"github.com/golang-jwt/jwt"
)

type GoogleTokenInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

type Claims struct {
	Sub string `json:"sub"`
	jwt.StandardClaims
}
