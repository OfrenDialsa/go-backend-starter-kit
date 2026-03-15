package repository

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type auditLogRepositoryImpl struct {
	db *database.WrapDB
}

func NewAuditLogRepository(db *database.WrapDB) AuditLogRepository {
	return &auditLogRepositoryImpl{db: db}
}

func (a *auditLogRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, audit *model.AuditLog) error {

	query := `
		INSERT INTO audit_logs (
			user_id, action, resource_type, resource_id, 
			details, ip_address, user_agent, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query,
			audit.UserID,
			audit.Action,
			audit.ResourceType,
			audit.ResourceID,
			audit.Details,
			audit.IPAddress,
			audit.UserAgent,
			audit.CreatedAt,
		).Scan(&audit.ID)
	} else {
		err = a.db.Database.Conn.QueryRow(ctx, query,
			audit.UserID,
			audit.Action,
			audit.ResourceType,
			audit.ResourceID,
			audit.Details,
			audit.IPAddress,
			audit.UserAgent,
			audit.CreatedAt,
		).Scan(&audit.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}
