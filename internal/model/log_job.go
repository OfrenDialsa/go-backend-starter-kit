package model

import (
	"time"
)

type LogJob struct {
	Id          string     `json:"id"`
	JobId       string     `json:"job_id"`
	Type        string     `json:"type"`
	Payload     []byte     `json:"payload"`
	Status      string     `json:"status"` // pending, processing, failed, completed
	RetryCount  int        `json:"retry_count"`
	LastError   *string    `json:"last_error"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
