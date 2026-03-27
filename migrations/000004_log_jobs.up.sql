CREATE TABLE IF NOT EXISTS log_jobs (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    last_error TEXT,
    scheduled_at TIMESTAMP DEFAULT now(),
    created_at TIMESTAMP DEFAULT now(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_log_jobs_status_retry ON log_jobs (status, retry_count);
CREATE INDEX idx_log_jobs_failed_only ON log_jobs (retry_count) WHERE status = 'failed';
CREATE INDEX idx_log_jobs_scheduled ON log_jobs (scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_log_jobs_type ON log_jobs (type);
CREATE INDEX idx_log_jobs_created_at ON log_jobs (created_at DESC);