package api

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
)

type Repositories struct {
	LogJob   repository.LogJobRepository
	User     repository.UserRepository
	Session  repository.SessionRepository
	Auditlog repository.AuditLogRepository
}

func NewRepositories(env *config.EnvironmentVariable, db *database.WrapDB) Repositories {
	return Repositories{
		LogJob:   repository.NewLogJobRepository(db),
		User:     repository.NewUserRepository(db),
		Session:  repository.NewSessionRepository(db),
		Auditlog: repository.NewAuditLogRepository(db),
	}
}
