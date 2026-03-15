package mailer

import "github/OfrenDialsa/go-gin-starter/internal/dto"

type Sender interface {
	Send(dto.MailgunRequest) (string, error)
}
