package mailer

import "github/OfrenDialsa/go-gin-starter/internal/dto"

type Sender interface {
	Send(dto.MailerRequest) (string, error)
}
