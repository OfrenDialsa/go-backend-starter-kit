package api

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/external"
	"github/OfrenDialsa/go-gin-starter/internal/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/service"
)

type Services struct {
	Producer service.ProducerService
	Consumer service.ConsumerService
	Auth     service.AuthService
	User     service.UserService
}

func NewServices(
	env *config.EnvironmentVariable,
	db *database.WrapDB,
	r Repositories,
	ext *external.ExternalService,
	mailer mailer.SmtpMailer,
	producerSvc service.ProducerService,
) Services {
	return Services{
		Consumer: service.NewConsumerService(env, mailer),
		Auth:     service.NewAuthService(env, db.Database.Conn, r.User, r.Session, producerSvc),
		User:     service.NewUserService(env, db.Database.Conn, r.User, r.Session, r.Auditlog, ext.Storage),
	}
}
