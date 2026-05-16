package api

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
)

type Repositories struct {
	User     repository.UserRepository
	Session  repository.SessionRepository
	Auditlog repository.AuditLogRepository
}

func NewRepositories(env *config.EnvironmentVariable, db *database.WrapDB) Repositories {
	return Repositories{
		User:     repository.NewUserRepository(db),
		Session:  repository.NewSessionRepository(db),
		Auditlog: repository.NewAuditLogRepository(db),
	}
}
