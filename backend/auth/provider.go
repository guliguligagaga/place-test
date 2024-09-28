package auth

import (
	"context"
	"errors"
)

type Provider interface {
	Name() string
	SignIn(context.Context, string) (string, error)
}

var providers = map[string]Provider{}

func RegisterProvider(p Provider) {
	providers[p.Name()] = p
}

func GetProvider(name string) (Provider, error) {
	p, ok := providers[name]
	if !ok {
		return nil, errors.New("unknown provider")
	}
	return p, nil
}
