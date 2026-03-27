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

func (s *consumerServiceImpl) ProcessEmail(ctx context.Context, msg *nsq.Message) (err error) {
	start := time.Now()
	var emailType string

	defer func() {
		status := "success"
		if err != nil {
			if msg.Attempts > 5 {
				status = "max_retry"
			} else {
				status = "failed"
			}
		}

		if emailType == "" {
			emailType = "unknown"
		}

		metrics.TrackEmailJob(emailType, status, time.Since(start))
	}()

	var payload dto.EmailTaskPayload
	if errUnmarshal := json.Unmarshal(msg.Body, &payload); errUnmarshal != nil {
		return nil
	}

	emailType = payload.Type
	jobId := payload.JobId
	var subject, body string

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
		if err == nil {
			err = fmt.Errorf("body is empty")
		}
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
		if jobId != "" {
			isPermanentError := payload.Email == ""

			if isPermanentError || msg.Attempts >= 5 {
				reason := "max retry reached"
				if isPermanentError {
					reason = "invalid recipient email (permanent error)"
				}

				log.Warn().Str("job_id", jobId).Msg(reason)
				s.logJobRepo.MarkAsFailed(ctx, nil, jobId, reason)
				return nil
			}

			errInc := s.logJobRepo.IncrementRetry(ctx, nil, jobId)
			errMark := s.logJobRepo.MarkAsFailed(ctx, nil, jobId, err.Error())

			event := log.Error().
				Err(err).
				Str("job_id", jobId).
				Uint16("attempt", msg.Attempts)

			if errInc != nil || errMark != nil {
				event.Interface("db_error", map[string]string{
					"increment": fmt.Sprint(errInc),
					"mark":      fmt.Sprint(errMark),
				}).Msg("failed to process email task and failed to update DB log")
			} else {
				event.Msg("failed to process email task")
			}
		}
		return err
	}

	if jobId != "" {
		err = s.logJobRepo.MarkAsCompleted(ctx, nil, jobId)
		if err != nil {
			return nil
		}
	}

	return nil
}
