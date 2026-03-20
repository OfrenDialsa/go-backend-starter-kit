package dto

type EmailTaskPayload struct {
	JobId string `json:"job_id"`
	Type  string `json:"type"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Link  string `json:"link,omitempty"`
}
