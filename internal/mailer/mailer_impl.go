package mailer

import (
	"fmt"

	"github/OfrenDialsa/go-gin-starter/internal/dto"

	"github.com/rs/zerolog/log"

	"github.com/mailgun/mailgun-go"
)

type MailgunClient interface {
	NewMessage(from, subject, text string, to ...string) *mailgun.Message
	Send(m *mailgun.Message) (string, string, error)
}

type MailgunMailer struct {
	Client       MailgunClient
	MailFrom     string
	MailFromName string
}

func NewMailgunMailer(client MailgunClient, mailFrom, mailFromName string) *MailgunMailer {
	return &MailgunMailer{
		Client:       client,
		MailFrom:     mailFrom,
		MailFromName: mailFromName,
	}
}

func (s *MailgunMailer) Send(req dto.MailgunRequest) (string, error) {
	from := fmt.Sprintf("%s <%s>", s.MailFromName, s.MailFrom)
	m := s.Client.NewMessage(from, req.Subject, "", req.To...)
	m.SetHtml(req.Body)

	for _, val := range req.Attachments {
		m.AddAttachment(val)
	}

	_, _, err := s.Client.Send(m)
	if err != nil {
		log.Error().Err(err).Msg("Error in sending mail")
		return err.Error(), err
	}

	log.Info().Str("Subject", req.Subject).Str("Recipient", req.To[0]).Msg(`Mail sent successfully to with subject`)

	return "", nil
}
