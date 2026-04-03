package consumer

import (
	"context"
	"time"

	"github/OfrenDialsa/go-gin-starter/cmd/api"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/service"

	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
)

type EmailConsumer struct {
	Env             *config.EnvironmentVariable
	ConsumerService service.ConsumerService
	Repository      api.Repositories
}

func NewEmailConsumer(
	env *config.EnvironmentVariable,
	cs service.ConsumerService,
	r api.Repositories,
) *EmailConsumer {
	return &EmailConsumer{
		Env:             env,
		ConsumerService: cs,
		Repository:      r,
	}
}

func (h *EmailConsumer) StartListen() error {
	configNSQ := nsq.NewConfig()

	topic := h.Env.MessageQueue.NSQ.Consumer.Email.Topic.SendEmail.TopicName
	channel := h.Env.MessageQueue.NSQ.Consumer.Email.ChannelName
	address := h.Env.MessageQueue.NSQ.Host

	consumer, err := nsq.NewConsumer(topic, channel, configNSQ)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", topic).
			Msg("[x] Failed to create email consumer")
		return err
	}

	consumer.AddHandler(h)

	err = consumer.ConnectToNSQD(address)
	if err != nil {
		log.Fatal().Err(err).Str("address", address).Msg("Failed to connect to nsqd")
		return err
	}

	log.Info().
		Str("topic", topic).
		Str("channel", channel).
		Msg("[v] Email Consumer is listening")

	return nil
}

func (h *EmailConsumer) HandleMessage(msg *nsq.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err := h.ConsumerService.HandleEvent(ctx, msg)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to process email message")
		return err
	}

	return nil
}
