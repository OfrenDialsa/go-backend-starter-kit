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
func TestPublishEvent(t *testing.T) {

	d := setupProducerService(t)
	topic := d.svc.(*service.ProducerServiceImpl).Env.MessageQueue.NSQ.Producer.Topic.SendEmail.TopicName

	tests := []struct {
		name      string
		event     dto.DomainEvent
		setupMock func(d *producerServiceTestDeps)
		wantErr   bool
	}{
		{
			name: "Success_PublishVerifyEmail",
			event: dto.DomainEvent{
				EventId:   "job-1",
				EventType: "user_registered",
				Payload:   []byte(`{"email":"ofren@example.com","name":"Ofren"}`),
			},
			setupMock: func(d *producerServiceTestDeps) {
				d.nsqClient.On("Publish", topic, mock.MatchedBy(func(body []byte) bool {
					var e dto.DomainEvent
					err := json.Unmarshal(body, &e)
					// Verifikasi bahwa yang di-publish adalah DomainEvent yang benar
					return err == nil && e.EventType == "user_registered" && e.EventId == "job-1"
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success_PublishForgotPassword",
			event: dto.DomainEvent{
				EventId:   "job-2",
				EventType: "forgot_password",
				Payload:   []byte(`{"email":"user@example.com"}`),
			},
			setupMock: func(d *producerServiceTestDeps) {
				d.nsqClient.On("Publish", topic, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_NSQConnectionRefused",
			event: dto.DomainEvent{
				EventType: "any_event",
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
			// Setup fresh mocks for each sub-test
			deps := setupProducerService(t)
			tt.setupMock(deps)

			// Panggil dengan argument event
			err := deps.svc.PublishEvent(tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "nsqd: connection refused")
			} else {
				assert.NoError(t, err)
			}

			deps.nsqClient.AssertExpectations(t)
		})
	}
}
