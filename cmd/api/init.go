package api

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/database/postgres"
	"github/OfrenDialsa/go-gin-starter/internal/infra/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/infra/storage"
	"github/OfrenDialsa/go-gin-starter/middleware"
	"github/OfrenDialsa/go-gin-starter/router"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nsqio/go-nsq"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

type Setup struct {
	Router     *gin.Engine
	Service    Services
	Repository Repositories
	WrapDB     *database.WrapDB
	Producer   *nsq.Producer
	Mailer     mailer.MailerService
}

func Init(env *config.EnvironmentVariable, wrapDB *database.WrapDB) (*Setup, error) {
	db := postgres.InitDatabase(env)
	mailer := mailer.NewMailerService(env, env.Mail.From, env.Mail.FromName)
	storage, err := storage.NewStorageService(context.Background(), env)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize storage service")
		return nil, err
	}

	repository := NewRepositories(env, wrapDB)
	service := NewServices(env, wrapDB, repository, storage, mailer, nil)
	handlers := NewHandlers(env, service, repository)

	// clean rate limit cache
	cCache := cache.New(5*time.Minute, 10*time.Minute)
	mw := middleware.NewMiddleware(env, repository.User, repository.Session, cCache)

	r := router.Handler{
		Env:         env,
		Middleware:  mw,
		AuthHandler: handlers.Auth,
		UserHandler: handlers.User,
		DB:          db,
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
