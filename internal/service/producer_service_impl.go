package service

import (
	"encoding/json"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"

	"github.com/rs/zerolog/log"
)

type ProducerTopic struct {
	SendEmailRequest string
}

type ProducerServiceImpl struct {
	Env         *config.EnvironmentVariable
	NsqProducer NsqClient
}

func NewProducerServiceImpl(
	env *config.EnvironmentVariable,
	nsqProducer NsqClient,
) ProducerService {
	return &ProducerServiceImpl{
		Env:         env,
		NsqProducer: nsqProducer,
	}
}

func (c *ProducerServiceImpl) PublishEvent(event dto.DomainEvent) error {
	message, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("json.Marshal failed for DomainEvent")
		return err
	}

	topic := c.Env.MessageQueue.NSQ.Producer.Topic.SendEmail.TopicName

	log.Debug().
		Str("topic", topic).
		Str("event_type", event.EventType).
		RawJSON("payload", message).
		Msg("[>>] Outgoing Domain Event")

	err = c.NsqProducer.Publish(topic, message)
	if err != nil {
		log.Error().Err(err).
			Str("topic", topic).
			Str("event_type", event.EventType).
			Msg("NsqProducer.Publish event failed")
		return err
	}

	return nil
}
