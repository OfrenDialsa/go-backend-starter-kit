package tests

import (
	"context"
	"encoding/json"
	"errors"
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

	createEventBody := func(eventType string, jobId string, payload interface{}) []byte {
		pBytes, _ := json.Marshal(payload)
		event := dto.DomainEvent{
			EventId:   jobId,
			EventType: eventType,
			Payload:   pBytes,
		}
		body, _ := json.Marshal(event)
		return body
	}

	tests := []struct {
		name      string
		body      []byte
		setupMock func(d *consumerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_UserRegistered",
			body: createEventBody(lib.NSQ_USER_REGISTERED_EVENT, "job-123", dto.EmailSendPayload{
				Email: "ofren@example.com",
				Name:  "Ofren",
				Token: "token-abc",
			}),
			setupMock: func(d *consumerServiceTestDeps) {
				// 1. Tambahkan Mock untuk Idempotency Lock
				d.logJobRepo.On("UpdateStatusToProcessing", ctx, nil, "job-123").Return(int64(1), nil)

				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectRegister && req.To[0] == "ofren@example.com"
				})).Return("msg-id", nil)

				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Skip_AlreadyProcessed_Idempotent",
			body: createEventBody(lib.NSQ_USER_REGISTERED_EVENT, "job-duplicate", dto.EmailSendPayload{
				Email: "ofren@example.com",
			}),
			setupMock: func(d *consumerServiceTestDeps) {
				// Return 0 artinya job sudah processing atau completed
				d.logJobRepo.On("UpdateStatusToProcessing", ctx, nil, "job-duplicate").Return(int64(0), nil)

				// Mailer TIDAK BOLEH dipanggil
				d.mailer.AssertNotCalled(t, "Send", mock.Anything)
			},
			wantErr: false,
		},
		{
			name: "Error_SMTPDown_ShouldRetry",
			body: createEventBody(lib.NSQ_PASSWORD_RESET_REQUESTED_EVENT, "job-retry", dto.EmailSendPayload{
				Email: "user@example.com",
				Token: "reset-123",
			}),
			setupMock: func(d *consumerServiceTestDeps) {
				d.logJobRepo.On("UpdateStatusToProcessing", ctx, nil, "job-retry").Return(int64(1), nil)

				d.mailer.On("Send", mock.Anything).Return("", errors.New("smtp connection refused"))

				d.logJobRepo.On("IncrementRetry", ctx, nil, "job-retry").Return(nil)
				d.logJobRepo.On("MarkAsFailed", ctx, nil, "job-retry", "smtp connection refused").Return(nil)
			},
			wantErr: true,
		},
		{
			name: "Error_EmptyEmail_PermanentFailure",
			body: createEventBody(lib.NSQ_USER_REGISTERED_EVENT, "job-perm-fail", dto.EmailSendPayload{
				Email: "",
				Name:  "Ofren",
			}),
			setupMock: func(d *consumerServiceTestDeps) {
				d.logJobRepo.On("UpdateStatusToProcessing", ctx, nil, "job-perm-fail").Return(int64(1), nil)

				// Di kode terbaru, email == "" dicek SEBELUM mailer.Send
				d.logJobRepo.On("MarkAsFailed", ctx, nil, "job-perm-fail", "invalid recipient email (permanent error)").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupConsumerService(t)
			tt.setupMock(d)

			message := &nsq.Message{Body: tt.body, Attempts: 1}
			err := d.svc.HandleEvent(ctx, message)

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

	// Helper untuk membuat []byte dari payload apapun
	marshalPayload := func(p interface{}) []byte {
		b, _ := json.Marshal(p)
		return b
	}

	tests := []struct {
		name      string
		event     dto.DomainEvent
		setupMock func(d *consumerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_VerifyEmailSuccess",
			event: dto.DomainEvent{
				EventId:   "job-abc",
				EventType: lib.NSQ_EMAIL_VERIFIED_EVENT,
				Payload: marshalPayload(dto.EmailSuccessPayload{
					Email: "ofren@example.com",
					Name:  "Ofren Dialsa",
				}),
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
			event: dto.DomainEvent{
				EventId:   "job-pwd",
				EventType: lib.NSQ_PASSWORD_RESET_REQUESTED_EVENT,
				Payload: marshalPayload(dto.EmailSendPayload{
					Email: "ofren@example.com",
					Name:  "Ofren",
					Token: "asdnaiofbnailjfnakJfe",
				}),
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
			event: dto.DomainEvent{
				EventId:   "job-pwd-ok",
				EventType: lib.NSQ_PASSWORD_RESET_SUCCESS_EVENT,
				Payload: marshalPayload(dto.EmailSuccessPayload{
					Email: "ofren@example.com",
					Name:  "Ofren",
				}),
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
			event: dto.DomainEvent{
				EventType: "random_type_123",
				Payload:   []byte(`{}`),
			},
			setupMock: func(d *consumerServiceTestDeps) {},
			wantErr:   false,
		},
		{
			name: "Success_Register_FirstTime",
			event: dto.DomainEvent{
				EventId:   "job-reg-1",
				EventType: lib.NSQ_USER_REGISTERED_EVENT,
				Payload: marshalPayload(dto.EmailSendPayload{
					Email: "ofren@example.com",
					Name:  "Ofren",
					Token: "token-123",
				}),
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectRegister
				})).Return("msg-123", nil)
				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-reg-1").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_ResendVerification",
			event: dto.DomainEvent{
				EventId:   "job-resend-123",
				EventType: lib.NSQ_RESEND_VERIFICATION_EVENT,
				Payload: marshalPayload(dto.EmailSendPayload{
					Email: "ofren@example.com",
					Name:  "Ofren",
					Token: "new-token-456",
				}),
			},
			setupMock: func(d *consumerServiceTestDeps) {
				d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
					return req.Subject == lib.DefaultEmailSubjectResend &&
						req.To[0] == "ofren@example.com"
				})).Return("msg-resend-id", nil)

				d.logJobRepo.On("MarkAsCompleted", ctx, nil, "job-resend-123").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupConsumerService(t)

			if tt.name != "Success_UnknownType_ShouldReturnNil" {
				d.logJobRepo.On("UpdateStatusToProcessing", ctx, nil, tt.event.EventId).Return(int64(1), nil)
			}

			tt.setupMock(d)

			body, _ := json.Marshal(tt.event)
			message := &nsq.Message{Body: body, Attempts: 1}

			err := d.svc.HandleEvent(ctx, message)

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
