package repository

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type LogJobRepository interface {
	Create(ctx context.Context, tx pgx.Tx, job *model.LogJob) error
	UpdateStatusToProcessing(ctx context.Context, tx pgx.Tx, jobId string) (int64, error)
	FindByJobId(ctx context.Context, tx pgx.Tx, jobId string) (*model.LogJob, error)
	MarkAsCompleted(ctx context.Context, tx pgx.Tx, jobId string) error
	MarkAsFailed(ctx context.Context, tx pgx.Tx, jobId string, errMsg string) error
	GetFailedForRetry(ctx context.Context, limit int) ([]model.LogJob, error)
	IncrementRetry(ctx context.Context, tx pgx.Tx, jobId string) error
}
