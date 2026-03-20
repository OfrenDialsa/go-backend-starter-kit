package tests

import (
	"context"
	"encoding/json"
	"errors"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/tests/mocks"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type consumerTestDeps struct {
	mailer     *mocks.Mailer
	logJobRepo mocks.LogJobRepository
	svc        service.ConsumerService
}

func setupConsumerService(t *testing.T) *consumerTestDeps {
	t.Helper()

	mockMailer := mocks.NewMailer(t)

	d := &consumerTestDeps{
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

// ===================== ProcessEmail Tests =====================
func TestProcessEmail_VerifyEmail(t *testing.T) {
	d := setupConsumerService(t)
	ctx := context.Background()

	payload := dto.EmailTaskPayload{
		Type:  "verify_email",
		Email: "ofrendialsa.dev@gmail.com",
		Name:  "Ofren Dialsa",
		Link:  "https://example.com/verify",
	}
	payloadBytes, _ := json.Marshal(payload)
	message := &nsq.Message{Body: payloadBytes}

	d.mailer.On("Send", mock.MatchedBy(func(req dto.MailerRequest) bool {
		return req.To[0] == payload.Email && req.Subject != ""
	})).Return("msg-id-123", nil)

	err := d.svc.ProcessEmail(ctx, message)

	assert.NoError(t, err)
	d.mailer.AssertExpectations(t)
}

func TestProcessEmail_InvalidJSON(t *testing.T) {
	d := setupConsumerService(t)
	ctx := context.Background()

	message := &nsq.Message{Body: []byte("invalid-payload-data")}

	err := d.svc.ProcessEmail(ctx, message)

	assert.NoError(t, err)
	d.mailer.AssertNotCalled(t, "Send", mock.Anything)
}

func TestProcessEmail_MailerError(t *testing.T) {
	d := setupConsumerService(t)
	ctx := context.Background()

	payload := dto.EmailTaskPayload{
		Type:  "verify_email",
		Email: "user@example.com",
	}
	payloadBytes, _ := json.Marshal(payload)
	message := &nsq.Message{Body: payloadBytes}

	d.mailer.On("Send", mock.Anything).Return("", errors.New("smtp connection timeout"))

	err := d.svc.ProcessEmail(ctx, message)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp connection timeout")
}

func TestProcessEmail_UnknownType(t *testing.T) {
	d := setupConsumerService(t)
	ctx := context.Background()

	payload := dto.EmailTaskPayload{
		Type: "unknown_action_type",
	}
	payloadBytes, _ := json.Marshal(payload)
	message := &nsq.Message{Body: payloadBytes}

	err := d.svc.ProcessEmail(ctx, message)

	assert.NoError(t, err)
	d.mailer.AssertNotCalled(t, "Send", mock.Anything)
}
