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

type producerTestDeps struct {
	nsqClient *mocks.NsqClient
	svc       service.ProducerService
}

func setupProducerService(t *testing.T) *producerTestDeps {
	t.Helper()

	mockNsq := mocks.NewNsqClient(t)

	d := &producerTestDeps{
		nsqClient: mockNsq,
	}

	d.svc = service.NewProducerServiceImpl(
		testEnv(),
		d.nsqClient,
	)

	return d
}

// ===================== SendEmailRequest Tests =====================

func TestSendEmailRequest_Success(t *testing.T) {
	d := setupProducerService(t)

	payload := dto.EmailTaskPayload{
		Type:  "verify_email",
		Email: "ofren.dialsa@example.com",
		Name:  "Ofren Dialsa",
		Link:  "https://example.com/verify",
	}

	d.nsqClient.On("Publish", "test-email-topic", mock.MatchedBy(func(body []byte) bool {
		var p dto.EmailTaskPayload
		err := json.Unmarshal(body, &p)
		return err == nil && p.Email == payload.Email && p.Type == payload.Type
	})).Return(nil)

	err := d.svc.SendEmailRequest(payload)

	assert.NoError(t, err)
	d.nsqClient.AssertExpectations(t)
}

func TestSendEmailRequest_PublishError(t *testing.T) {
	d := setupProducerService(t)

	payload := dto.EmailTaskPayload{
		Type:  "forgot_password",
		Email: "user@example.com",
	}

	d.nsqClient.On("Publish", "test-email-topic", mock.Anything).
		Return(errors.New("nsqd: connection refused"))

	err := d.svc.SendEmailRequest(payload)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nsqd: connection refused")
	d.nsqClient.AssertExpectations(t)
}

func TestSendEmailRequest_MarshalCheck(t *testing.T) {
	d := setupProducerService(t)

	payload := dto.EmailTaskPayload{}

	d.nsqClient.On("Publish", mock.Anything, mock.Anything).Return(nil)

	err := d.svc.SendEmailRequest(payload)

	assert.NoError(t, err)
	d.nsqClient.AssertExpectations(t)
}
