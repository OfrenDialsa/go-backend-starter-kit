package repository

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type AuditLogRepository interface {
	Create(ctx context.Context, tx pgx.Tx, audit *model.AuditLog) error
}
