package dto

type EmailTaskPayload struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Link  string `json:"link,omitempty"`
}
