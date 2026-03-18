package service

import "github/OfrenDialsa/go-gin-starter/internal/dto"

type ProducerService interface {
	SendEmailRequest(payload dto.EmailTaskPayload) error
}
