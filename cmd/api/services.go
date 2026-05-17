package api

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/infra/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/infra/storage"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"sync"
)

type Services struct {
	Auth service.AuthService
	User service.UserService
}

func NewServices(
	env *config.EnvironmentVariable,
	db *database.WrapDB,
	r Repositories,
	st storage.StorageService,
	mailer mailer.SmtpMailer,
	wg *sync.WaitGroup,
) Services {
	return Services{
		Auth: service.NewAuthService(env, db.Database.Conn, mailer, r.User, r.Session, wg),
		User: service.NewUserService(env, db.Database.Conn, r.User, r.Session, r.Auditlog, st),
	}
}
