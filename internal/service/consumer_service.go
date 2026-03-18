package service

import (
	"context"

	"github.com/nsqio/go-nsq"
)

type ConsumerService interface {
	ProcessEmail(ctx context.Context, message *nsq.Message) error
}
