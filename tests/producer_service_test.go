package tests

import (
	"encoding/json"
	"errors"
	"testing"

	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type producerServiceTestDeps struct {
	nsqClient *mocks.NsqClient
	svc       service.ProducerService
}

func setupProducerService(t *testing.T) *producerServiceTestDeps {
	t.Helper()

	mockNsq := mocks.NewNsqClient(t)

	d := &producerServiceTestDeps{
		nsqClient: mockNsq,
	}

	d.svc = service.NewProducerServiceImpl(
		testEnv(),
		d.nsqClient,
	)

	return d
}

func TestSendEmailRequest(t *testing.T) {
	topic := "test-email-topic"

	tests := []struct {
		name      string
		payload   dto.EmailTaskPayload
		setupMock func(d *producerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_PublishVerifyEmail",
			payload: dto.EmailTaskPayload{
				Type:  "verify_email",
				Email: "ofren.dialsa@example.com",
				Name:  "Ofren Dialsa",
				Link:  "https://example.com/verify",
			},
			setupMock: func(d *producerServiceTestDeps) {
				d.nsqClient.On("Publish", topic, mock.MatchedBy(func(body []byte) bool {
					var p dto.EmailTaskPayload
					err := json.Unmarshal(body, &p)
					return err == nil && p.Email == "ofren.dialsa@example.com" && p.Type == "verify_email"
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_PublishForgotPassword",
			payload: dto.EmailTaskPayload{
				Type:  "forgot_password",
				Email: "user@example.com",
			},
			setupMock: func(d *producerServiceTestDeps) {
				d.nsqClient.On("Publish", topic, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_NSQConnectionRefused",
			payload: dto.EmailTaskPayload{
				Type: "any_type",
			},
			setupMock: func(d *producerServiceTestDeps) {
				d.nsqClient.On("Publish", topic, mock.Anything).
					Return(errors.New("nsqd: connection refused"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupProducerService(t)
			tt.setupMock(d)

			err := d.svc.SendEmailRequest(tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "nsqd: connection refused")
			} else {
				assert.NoError(t, err)
			}

			d.nsqClient.AssertExpectations(t)
		})
	}
}
