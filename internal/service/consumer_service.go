package service

import (
	"context"

	"github.com/nsqio/go-nsq"
)

type ConsumerService interface {
	HandleEvent(ctx context.Context, msg *nsq.Message) error
}
