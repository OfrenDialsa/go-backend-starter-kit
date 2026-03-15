package api

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/external"
	"github/OfrenDialsa/go-gin-starter/internal/mailer"
	"github/OfrenDialsa/go-gin-starter/middleware"
	"github/OfrenDialsa/go-gin-starter/router"

	"github.com/gin-gonic/gin"
	"github.com/mailgun/mailgun-go"
	"github.com/rs/zerolog/log"
)

type Setup struct {
	Router     *gin.Engine
	Service    Services
	Repository Repositories
	WrapDB     *database.WrapDB
}

func Init(env *config.EnvironmentVariable, wrapDB *database.WrapDB) (*Setup, error) {

	mg := mailgun.NewMailgun(env.Mail.Mailgun.Domain, env.Mail.Mailgun.ApiKey)
	sender := mailer.NewMailgunMailer(mg, env.Mail.From, env.Mail.FromName)
	repository := NewRepositories(env, wrapDB)

	extService, err := external.NewExternalService(env)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize external service")
		return nil, err
	}

	service := NewServices(env, wrapDB, repository, extService, sender)

	handlers := NewHandlers(env, service, repository)

	mw := middleware.NewMiddleware(env, repository.User, repository.Session)

	r := router.Handler{
		Env:         env,
		Middleware:  mw,
		AuthHandler: handlers.Auth,
		UserHandler: handlers.User,
	}

	routes := router.NewRouter(env, r)

	return &Setup{
		Router:     routes,
		Repository: repository,
		Service:    service,
		WrapDB:     wrapDB,
	}, nil
}
