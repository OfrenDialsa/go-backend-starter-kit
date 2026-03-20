package repository

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type sessionRepositoryImpl struct {
	db *database.WrapDB
}

func NewSessionRepository(db *database.WrapDB) SessionRepository {
	return &sessionRepositoryImpl{db: db}
}

func (r *sessionRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, session *model.UserSession) error {
	query := `
        INSERT INTO user_sessions (
            session_id, user_id, token_hash, type, 
            ip_address, user_agent, expires_at, created_at, updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `

	var runner interface {
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	} = r.db.Database.Conn

	if tx != nil {
		runner = tx
	}

	err := runner.QueryRow(ctx, query,
		session.SessionId,
		session.UserId,
		session.TokenHash,
		session.Type,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	).Scan(&session.Id)

	return err
}

func (r *sessionRepositoryImpl) Update(ctx context.Context, session *model.UserSession) error {
	query := `
		UPDATE user_sessions 
		SET token_hash = $1, 
		    expires_at = $2, 
		    updated_at = $3, 
		    ip_address = $4, 
		    user_agent = $5
		WHERE session_id = $6 AND revoked_at IS NULL
	`
	_, err := r.db.Database.Conn.Exec(ctx, query,
		session.TokenHash,
		session.ExpiresAt,
		session.UpdatedAt,
		session.IPAddress,
		session.UserAgent,
		session.SessionId,
	)
	return err
}
func (r *sessionRepositoryImpl) GetByToken(ctx context.Context, tokenHash string, tokenType string) (*model.UserSession, error) {
	query := `
        SELECT id, session_id, user_id, token_hash, type, ip_address, user_agent, expires_at, created_at, updated_at, revoked_at
        FROM user_sessions
        WHERE token_hash = $1 AND type = $2
    `

	var session model.UserSession
	err := r.db.Database.Conn.QueryRow(ctx, query, tokenHash, tokenType).Scan(
		&session.Id,
		&session.SessionId,
		&session.UserId,
		&session.TokenHash,
		&session.Type,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.RevokedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}
	return &session, nil
}

func (r *sessionRepositoryImpl) GetBySessionId(ctx context.Context, sessionID string) (*model.UserSession, error) {
	query := `
        SELECT id, session_id, user_id, token_hash, type, ip_address, user_agent, expires_at, created_at, updated_at, revoked_at
        FROM user_sessions
        WHERE session_id = $1
    `

	var session model.UserSession
	err := r.db.Database.Conn.QueryRow(ctx, query, sessionID).Scan(
		&session.Id,
		&session.SessionId,
		&session.UserId,
		&session.TokenHash,
		&session.Type,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.RevokedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepositoryImpl) RevokeByToken(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1
	`
	_, err := r.db.Database.Conn.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

func (r *sessionRepositoryImpl) RevokeBySessionId(ctx context.Context, sessionID string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE session_id = $1
	`
	_, err := r.db.Database.Conn.Exec(ctx, query, sessionID)
	return err
}

func (r *sessionRepositoryImpl) RevokeAllUserSessions(ctx context.Context, tx pgx.Tx, userID string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, userID)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, userID)
	}
	if err != nil {
		return fmt.Errorf("failed to revoke all user sessions: %w", err)
	}

	return nil
}

func (r *sessionRepositoryImpl) DeleteSession(ctx context.Context, tx pgx.Tx, sessionId string) error {
	query := `
		DELETE FROM user_sessions
		WHERE session_id = $1
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, sessionId)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, sessionId)
	}

	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (r *sessionRepositoryImpl) RevokeAllExcept(ctx context.Context, userID string, currentSessionID string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 
		  AND session_id != $2 
		  AND revoked_at IS NULL
	`
	_, err := r.db.Database.Conn.Exec(ctx, query, userID, currentSessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke other sessions: %w", err)
	}

	return nil
}
