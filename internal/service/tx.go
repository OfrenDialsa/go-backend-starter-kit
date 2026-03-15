package service

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TxStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
