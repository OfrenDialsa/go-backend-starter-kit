package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/metrics"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
)

type consumerServiceImpl struct {
	env        *config.EnvironmentVariable
	logJobRepo repository.LogJobRepository
	mailer     mailer.SmtpMailer
}

func NewConsumerService(
	env *config.EnvironmentVariable,
	logJobRepo repository.LogJobRepository,
	mailer mailer.SmtpMailer,
) ConsumerService {
	return &consumerServiceImpl{
		env:        env,
		logJobRepo: logJobRepo,
		mailer:     mailer,
	}
}

func (s *consumerServiceImpl) HandleEvent(ctx context.Context, msg *nsq.Message) error {
	var event dto.DomainEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return nil
	}

	handlers := map[string]func(context.Context, dto.DomainEvent, *nsq.Message) error{
		lib.NSQ_USER_REGISTERED_EVENT:          s.handleUserRegistered,
		lib.NSQ_RESEND_VERIFICATION_EVENT:      s.handleResendVerification,
		lib.NSQ_PASSWORD_RESET_REQUESTED_EVENT: s.handlePasswordReset,
		lib.NSQ_PASSWORD_RESET_SUCCESS_EVENT:   s.handlePasswordResetSuccess,
		lib.NSQ_EMAIL_VERIFIED_EVENT:           s.handleEmailVerified,
	}

	handler, err := handlers[event.EventType]
	if !err {
		log.Warn().Str("event_type", event.EventType).Msg("[x]unknown event")
		return nil
	}

	return handler(ctx, event, msg)
}

func (s *consumerServiceImpl) handleUserRegistered(ctx context.Context, event dto.DomainEvent, msg *nsq.Message) error {
	var payload dto.EmailSendPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	verifLink := fmt.Sprintf("%s?token=%s", s.env.External.VerifyEmailURL, payload.Token)

	subject := lib.DefaultEmailSubjectRegister
	body, err := lib.BuildEmailBodyRegister(payload.Name, verifLink)
	if err != nil || body == "" {
		return fmt.Errorf("failed to build email body")
	}

	return s.sendEmail(ctx, event.EventId, payload.Email, subject, body, event.EventType, msg.Attempts)
}

func (s *consumerServiceImpl) handleResendVerification(ctx context.Context, event dto.DomainEvent, msg *nsq.Message) error {
	var payload dto.EmailSendPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	verifLink := fmt.Sprintf("%s?token=%s", s.env.External.VerifyEmailURL, payload.Token)

	subject := lib.DefaultEmailSubjectResend
	body, err := lib.BuildEmailBodyResendVerification(payload.Name, verifLink)
	if err != nil || body == "" {
		return fmt.Errorf("failed to build email body")
	}

	return s.sendEmail(ctx, event.EventId, payload.Email, subject, body, event.EventType, msg.Attempts)
}

func (s *consumerServiceImpl) handleEmailVerified(ctx context.Context, event dto.DomainEvent, msg *nsq.Message) error {
	var payload dto.EmailSuccessPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	loginLink := s.env.External.FrontendURL + "/login"

	subject := lib.DefaultEmailSubjectVerifyEmailSuccess
	body, err := lib.BuildEmailBodyVerifyEmailSuccess(payload.Name, loginLink)
	if err != nil || body == "" {
		return fmt.Errorf("failed to build email body")
	}

	return s.sendEmail(ctx, event.EventId, payload.Email, subject, body, event.EventType, msg.Attempts)
}

func (s *consumerServiceImpl) handlePasswordReset(ctx context.Context, event dto.DomainEvent, msg *nsq.Message) error {
	var payload dto.EmailSendPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	resetPassLink := fmt.Sprintf("%s?token=%s", s.env.External.ResetPasswordURL, payload.Token)
	subject := lib.DefaultEmailSubjectResetPassword
	body, err := lib.BuildEmailBodyResetPassword(payload.Name, resetPassLink)
	if err != nil || body == "" {
		return fmt.Errorf("failed to build email body")
	}

	return s.sendEmail(ctx, event.EventId, payload.Email, subject, body, event.EventType, msg.Attempts)
}

func (s *consumerServiceImpl) handlePasswordResetSuccess(ctx context.Context, event dto.DomainEvent, msg *nsq.Message) error {
	var payload dto.EmailSuccessPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	subject := lib.DefaultEmailSubjectPasswordResetSuccess
	body, err := lib.BuildEmailBodyPasswordResetSuccess(payload.Name)
	if err != nil || body == "" {
		return fmt.Errorf("failed to build email body")
	}

	return s.sendEmail(ctx, event.EventId, payload.Email, subject, body, event.EventType, msg.Attempts)
}

func (s *consumerServiceImpl) sendEmail(ctx context.Context, jobId string, email string, subject string, body string, eventType string, attempt uint16) (err error) {
	start := time.Now()

	if jobId != "" {
		affected, err := s.logJobRepo.UpdateStatusToProcessing(ctx, nil, jobId)
		if err != nil {
			return err
		}
		if affected == 0 {
			log.Info().Str("job_id", jobId).Msg("[>]skipping: job already processing or completed")
			return nil
		}
	}

	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
			if attempt > 5 {
				status = "max_retry"
			}
		}
		metrics.TrackEmailJob(eventType, status, time.Since(start))
	}()

	if email == "" {
		reason := "invalid recipient email (permanent error)"
		log.Warn().Str("job_id", jobId).Msg(reason)
		_ = s.logJobRepo.MarkAsFailed(ctx, nil, jobId, reason)
		return nil
	}

	mailData := dto.MailerRequest{To: []string{email}, Subject: subject, Body: body}
	_, err = s.mailer.Send(mailData)

	if err != nil {
		_ = s.logJobRepo.IncrementRetry(ctx, nil, jobId)
		_ = s.logJobRepo.MarkAsFailed(ctx, nil, jobId, err.Error())
		log.Error().Err(err).Str("job_id", jobId).Uint16("attempt", attempt).Msg("failed to send email")
		return err
	}

	if jobId != "" {
		if err := s.logJobRepo.MarkAsCompleted(ctx, nil, jobId); err != nil {
			log.Error().Err(err).Str("job_id", jobId).Msg("failed to mark job completed")
		}
	}

	return nil
}
