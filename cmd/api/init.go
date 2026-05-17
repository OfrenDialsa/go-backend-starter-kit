package api

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/infra/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/infra/storage"
	"github/OfrenDialsa/go-gin-starter/middleware"
	"github/OfrenDialsa/go-gin-starter/router"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
)

type Setup struct {
	Router     *gin.Engine
	Service    Services
	Repository Repositories
	WrapDB     *database.WrapDB
	Producer   *nsq.Producer
	Mailer     mailer.SmtpMailer
}

func Init(env *config.EnvironmentVariable, wrapDB *database.WrapDB) (*Setup, error) {
	mailer := mailer.NewSMTPMailer(env, env.Mail.From, env.Mail.FromName)
	repository := NewRepositories(env, wrapDB)
	storage, err := storage.New(context.Background(), env)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize storage service")
		return nil, err
	}

	service := NewServices(env, wrapDB, repository, storage, mailer, nil)

	handlers := NewHandlers(env, service, repository)

	mw := middleware.NewMiddleware(env, wrapDB, repository.User, repository.Session)

	if mwImpl, ok := mw.(*middleware.MiddlewareImpl); ok {
		mwImpl.CleanRateLimit(10*time.Minute, 30*time.Minute)
	}

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
		Mailer:     mailer,
	}, nil
}
