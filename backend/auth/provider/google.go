package provider

import (
	"context"
	"errors"
	"log"
	"os"

	"google.golang.org/api/idtoken"
)

type Google struct {
	clientId string
}

func NewGoogle() Google {
	return Google{
		clientId: os.Getenv("GOOGLE_CLIENT_ID"),
	}
}

func (p Google) Name() string {
	return "google"
}

func (p Google) SignIn(ctx context.Context, token string) (string, error) {
	payload, err := idtoken.Validate(ctx, token, p.clientId)
	if err != nil {
		log.Printf("Error validating Google token: %v", err)

		return "", errors.New("invalid token")
	}

	return payload.Subject, nil
}
