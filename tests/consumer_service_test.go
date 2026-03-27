package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/tests/mocks"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type consumerServiceTestDeps struct {
	mailer     *mocks.Mailer
	logJobRepo mocks.LogJobRepository
	svc        service.ConsumerService
}

func setupConsumerService(t *testing.T) *consumerServiceTestDeps {
	t.Helper()

	mockMailer := mocks.NewMailer(t)

	d := &consumerServiceTestDeps{
		logJobRepo: *mocks.NewLogJobRepository(t),
		mailer:     mockMailer,
	}

	d.svc = service.NewConsumerService(
		testEnv(),
		&d.logJobRepo,
		d.mailer,
	)

	return d
}

func TestProcessEmail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		payload   interface{}
		setupMock func(d *consumerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_VerifyEmail_WithJobLogging",
			payload: dto.EmailTaskPayload{
				JobId: "job-123",
				Type:  "verify_email",
				Email: "ofren@example.com",
				Name:  "Ofren",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("msg-id", nil)

				d.logJobRepo.On("MarkAsCompleted", ctx, mock.Anything, "job-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_EmptyEmailAddress",
			payload: dto.EmailTaskPayload{
				JobId: "job-empty-email",
				Type:  "verify_email",
				Email: "",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("", fmt.Errorf("empty email address"))

				d.logJobRepo.On("MarkAsFailed", ctx, nil, "job-empty-email", "invalid recipient email (permanent error)").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_MailerFailed_UpdateJobStatus",
			payload: dto.EmailTaskPayload{
				JobId: "job-failed",
				Type:  "forgot_password",
				Email: "user@example.com",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("", errors.New("smtp down"))

				// PERBAIKAN: Berdasarkan log, kodemu memanggil IncrementRetry
				d.logJobRepo.On("IncrementRetry", ctx, nil, "job-failed").Return(nil)

				// Dan memanggil MarkAsFailed
				d.logJobRepo.On("MarkAsFailed", ctx, nil, "job-failed", "smtp down").Return(nil)
			},
			wantErr: true,
		},
		{
			name: "Success_NoJobId_ShouldOnlySendEmail",
			payload: dto.EmailTaskPayload{
				JobId: "", // Edge case: Tidak ada JobId (mungkin request manual)
				Type:  "verify_email",
				Email: "user@example.com",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("msg-ok", nil)
				// logJobRepo tidak boleh dipanggil sama sekali
			},
			wantErr: false,
		},
		{
			name: "Error_RepoUpdateFailed_StillReturnSuccess",
			payload: dto.EmailTaskPayload{
				JobId: "job-repo-fail",
				Type:  "verify_email",
				Email: "ofren@example.com",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("msg-ok", nil)

				// Edge case: Email terkirim tapi database sedang sibuk/error
				d.logJobRepo.On("MarkAsCompleted", ctx, mock.Anything, "job-repo-fail").
					Return(errors.New("db dead"))
			},
			wantErr: false, // Tetap false karena prioritas utama (Email) sudah terkirim
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupConsumerService(t)
			tt.setupMock(d)

			var body []byte
			if b, ok := tt.payload.([]byte); ok {
				body = b
			} else {
				body, _ = json.Marshal(tt.payload)
			}

			// Mock NSQ Message
			message := &nsq.Message{Body: body}

			err := d.svc.ProcessEmail(ctx, message)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			d.mailer.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
		})
	}
}

func TestProcessEmail_PayloadTypes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		payload   dto.EmailTaskPayload
		setupMock func(d *consumerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_VerifyEmailSuccess",
			payload: dto.EmailTaskPayload{
				JobId: "job-abc",
				Type:  "verify_email_success",
				Email: "ofren@example.com",
				Name:  "Ofren Dialsa",
				Link:  "https://example.com/login",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectVerifyEmailSuccess
				})).Return("msg-id", nil)
				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-abc").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_ForgotPassword",
			payload: dto.EmailTaskPayload{
				JobId: "job-pwd",
				Type:  "forgot_password",
				Email: "ofren@example.com",
				Name:  "Ofren",
				Link:  "https://example.com/reset",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectResetPassword
				})).Return("msg-id", nil)
				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-pwd").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_PasswordResetSuccess",
			payload: dto.EmailTaskPayload{
				JobId: "job-pwd-ok",
				Type:  "password_reset_success",
				Email: "ofren@example.com",
				Name:  "Ofren",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectPasswordResetSuccess
				})).Return("msg-id", nil)
				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-pwd-ok").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_UnknownType_ShouldReturnNil",
			payload: dto.EmailTaskPayload{
				Type:  "random_type_123",
				Email: "ofren@example.com",
			},
			setupMock: func(d *consumerServiceTestDeps) {
			},
			wantErr: false,
		},
		{
			name: "Success_VerifyEmail_WithEmptyLink_StillSends",
			payload: dto.EmailTaskPayload{
				JobId: "job-empty-link",
				Type:  "verify_email",
				Email: "ofren@example.com",
				Name:  "Ofren",
				Link:  "",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.To[0] == "ofren@example.com" && req.Subject != ""
				})).Return("msg-123", nil)

				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-empty-link").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_ThanksEmail_NoLinkNeeded",
			payload: dto.EmailTaskPayload{
				JobId: "job-thanks",
				Type:  "password_reset_success",
				Email: "ofren@example.com",
				Name:  "Ofren",
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.Anything).Return("msg-456", nil)
				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-thanks").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupConsumerService(t)
			tt.setupMock(d)

			body, _ := json.Marshal(tt.payload)
			message := &nsq.Message{Body: body, Attempts: 1}

			err := d.svc.ProcessEmail(ctx, message)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			d.mailer.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
		})
	}
}
