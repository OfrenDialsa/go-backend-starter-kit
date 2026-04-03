package dto

import (
	"encoding/json"
	"time"
)

type EmailSendPayload struct {
	JobId  string `json:"job_id"`
	UserId string `json:"user_id"`
	Type   string `json:"type"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Token  string `json:"token,omitempty"`
}

type EmailSuccessPayload struct {
	UserId string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type DomainEvent struct {
	EventId    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}
