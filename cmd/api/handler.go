package api

import (
	"github/OfrenDialsa/go-gin-starter/config"

	"github/OfrenDialsa/go-gin-starter/internal/handler"
)

type Handlers struct {
	Auth handler.AuthHandler
	User handler.UserHandler
}

func NewHandlers(env *config.EnvironmentVariable, s Services, r Repositories) Handlers {
	return Handlers{
		Auth: handler.NewAuthHandler(env, s.Auth),
		User: handler.NewUserHandler(s.User),
	}
}
