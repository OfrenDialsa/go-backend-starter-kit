package service

import (
	"encoding/json"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"

	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
)

type ProducerTopic struct {
	SendEmailRequest string
}

type ProducerServiceImpl struct {
	Env         *config.EnvironmentVariable
	Topic       ProducerTopic
	NsqProducer *nsq.Producer
}

func NewProducerServiceImpl(
	env *config.EnvironmentVariable,
	nsqProducer *nsq.Producer,
) ProducerService {
	topics := ProducerTopic{
		SendEmailRequest: env.MessageQueue.NSQ.Producer.Topic.SendEmail.TopicName,
	}

	return &ProducerServiceImpl{
		Env:         env,
		Topic:       topics,
		NsqProducer: nsqProducer,
	}
}

func (c *ProducerServiceImpl) SendEmailRequest(payload dto.EmailTaskPayload) error {
	message, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("json.Marshal failed for EmailTaskPayload")
		return err
	}

	dst := c.Topic.SendEmailRequest
	log.Debug().
		Str("topic", dst).
		RawJSON("payload", message).
		Msg("[>>] Outgoing Email Message")

	err = c.NsqProducer.Publish(dst, message)
	if err != nil {
		log.Error().Err(err).Str("topic", dst).Msg("NsqProducer.Publish email failed")
		return err
	}

	return nil
}
