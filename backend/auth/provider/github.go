package provider

import (
	"context"
	"fmt"
	"os"
)

type GitHubProvider struct {
	clientID     string
	clientSecret string
}

func NewGitHub() *GitHubProvider {
	return &GitHubProvider{
		clientID:     os.Getenv("GITHUB_CLIENT_ID"),
		clientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
	}
}

func (p *GitHubProvider) Name() string {
	return "github"
}

func (p *GitHubProvider) SignIn(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
