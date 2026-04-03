package service

import "github/OfrenDialsa/go-gin-starter/internal/dto"

type ProducerService interface {
	PublishEvent(event dto.DomainEvent) error
}
