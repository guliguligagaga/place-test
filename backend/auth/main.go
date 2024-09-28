package auth

import (
	"auth/provider"
	"web"
)

func Run() {
	instance := web.MakeServer(web.WithGinEngine)
	instance.AppendGinRoutes(registerRoutes)
	RegisterProvider(provider.NewGoogle())
	RegisterProvider(provider.NewGitHub())

	instance.Run()
}
