package service

import (
	"context"
	"encoding/json"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/mailer"
	"github/OfrenDialsa/go-gin-starter/lib"

	"github.com/nsqio/go-nsq"
)

type consumerServiceImpl struct {
	env    *config.EnvironmentVariable
	mailer mailer.SmtpMailer
}

func NewConsumerService(env *config.EnvironmentVariable, mailer mailer.SmtpMailer) ConsumerService {
	return &consumerServiceImpl{
		env:    env,
		mailer: mailer,
	}
}

func (s *consumerServiceImpl) ProcessEmail(ctx context.Context, message *nsq.Message) error {
	var payload dto.EmailTaskPayload
	if err := json.Unmarshal(message.Body, &payload); err != nil {
		return nil
	}

	var subject, body string
	var err error

	switch payload.Type {
	case "verify_email":
		subject = lib.DefaultEmailSubject
		body, err = lib.BuildEmailBody(payload.Name, payload.Link)
	case "verify_email_success":
		subject = lib.DefaultEmailSubjectVerifyEmailSuccess
		body, err = lib.BuildEmailBodyVerifyEmailSuccess(payload.Name, payload.Link)
	case "forgot_password":
		subject = lib.DefaultEmailSubjectResetPassword
		body, err = lib.BuildEmailBodyResetPassword(payload.Name, payload.Link)
	case "password_reset_success":
		subject = lib.DefaultEmailSubjectPasswordResetSuccess
		body, err = lib.BuildEmailBodyPasswordResetSuccess(payload.Name)
	default:
		return nil
	}

	if err != nil || body == "" {
		return err
	}

	mailData := dto.MailerRequest{
		To:          []string{payload.Email},
		Subject:     subject,
		Body:        body,
		Attachments: []string{},
	}

	_, err = s.mailer.Send(mailData)
	if err != nil {
		return err
	}

	return nil
}
