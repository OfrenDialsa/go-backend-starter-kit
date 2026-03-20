package repository

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type SessionRepository interface {
	Create(ctx context.Context, tx pgx.Tx, session *model.UserSession) error
	Update(ctx context.Context, session *model.UserSession) error
	GetBySessionId(ctx context.Context, sessionId string) (*model.UserSession, error)
	GetByToken(ctx context.Context, tokenHash string, tokenType string) (*model.UserSession, error)
	DeleteSession(ctx context.Context, tx pgx.Tx, sessionId string) error
	RevokeBySessionId(ctx context.Context, sessionID string) error
	RevokeByToken(ctx context.Context, tokenHash string) error
	RevokeAllUserSessions(ctx context.Context, tx pgx.Tx, userId string) error
	RevokeAllExcept(ctx context.Context, userId string, sessionId string) error
}
