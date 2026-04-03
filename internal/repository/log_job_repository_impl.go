package repository

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type logJobRepositoryImpl struct {
	db *database.WrapDB
}

func NewLogJobRepository(db *database.WrapDB) LogJobRepository {
	return &logJobRepositoryImpl{db: db}
}

func (r *logJobRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, job *model.LogJob) error {
	query := `
		INSERT INTO log_jobs (
			job_id, type, payload, status, retry_count, scheduled_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			job.JobId,
			job.Type,
			job.Payload,
			job.Status,
			job.RetryCount,
			job.ScheduledAt,
			job.CreatedAt,
		)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query,
			job.JobId,
			job.Type,
			job.Payload,
			job.Status,
			job.RetryCount,
			job.ScheduledAt,
			job.CreatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create log job: %w", err)
	}

	return nil
}

func (r *logJobRepositoryImpl) UpdateStatusToProcessing(ctx context.Context, tx pgx.Tx, jobId string) (int64, error) {
	query := `
		UPDATE log_jobs
		SET status = 'processing'
		WHERE job_id = $1 AND status = 'pending'
	`

	var err error
	var rowsAffected int64

	if tx != nil {
		result, errExec := tx.Exec(ctx, query, jobId)
		if errExec == nil {
			rowsAffected = result.RowsAffected()
		}
		err = errExec
	} else {
		result, errExec := r.db.Database.Conn.Exec(ctx, query, jobId)
		if errExec == nil {
			rowsAffected = result.RowsAffected()
		}
		err = errExec
	}

	if err != nil {
		return 0, fmt.Errorf("failed to update status to processing: %w", err)
	}

	return rowsAffected, nil
}

func (r *logJobRepositoryImpl) FindByJobId(ctx context.Context, tx pgx.Tx, jobId string) (*model.LogJob, error) {
	query := `
		SELECT job_id, type, payload, status, retry_count, last_error, 
		       scheduled_at, completed_at, created_at
		FROM log_jobs
		WHERE job_id = $1
	`

	var job model.LogJob
	var err error

	if tx != nil {
		err = tx.QueryRow(ctx, query, jobId).Scan(
			&job.JobId,
			&job.Type,
			&job.Payload,
			&job.Status,
			&job.RetryCount,
			&job.LastError,
			&job.ScheduledAt,
			&job.CompletedAt,
			&job.CreatedAt,
		)
	} else {
		err = r.db.Database.Conn.QueryRow(ctx, query, jobId).Scan(
			&job.JobId,
			&job.Type,
			&job.Payload,
			&job.Status,
			&job.RetryCount,
			&job.LastError,
			&job.ScheduledAt,
			&job.CompletedAt,
			&job.CreatedAt,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find log job: %w", err)
	}

	return &job, nil
}

func (r *logJobRepositoryImpl) MarkAsCompleted(ctx context.Context, tx pgx.Tx, jobId string) error {
	query := `
		UPDATE log_jobs
		SET status = 'completed',
			completed_at = $2
		WHERE job_id = $1
	`

	now := time.Now()

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, jobId, now)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, jobId, now)
	}

	if err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	return nil
}

func (r *logJobRepositoryImpl) MarkAsFailed(ctx context.Context, tx pgx.Tx, jobId string, errMsg string) error {
	query := `
		UPDATE log_jobs
		SET status = 'failed',
			last_error = $2,
			retry_count = retry_count + 1
		WHERE job_id = $1
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, jobId, errMsg)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, jobId, errMsg)
	}

	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	return nil
}

func (r *logJobRepositoryImpl) GetFailedForRetry(ctx context.Context, limit int) ([]model.LogJob, error) {
	query := `
		SELECT job_id, type, payload, status, retry_count
		FROM log_jobs
		WHERE status = 'failed'
		AND retry_count < 3
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Database.Conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.LogJob

	for rows.Next() {
		var job model.LogJob
		err := rows.Scan(
			&job.JobId,
			&job.Type,
			&job.Payload,
			&job.Status,
			&job.RetryCount,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *logJobRepositoryImpl) IncrementRetry(ctx context.Context, tx pgx.Tx, jobId string) error {
	query := `
        UPDATE log_jobs
        SET retry_count = retry_count + 1
        WHERE job_id = $1
		RETURNING retry_count;
    `

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, jobId)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, jobId)
	}

	if err != nil {
		return fmt.Errorf("failed to increment retry: %w", err)
	}

	return nil
}
