package nsq

import (
	"github/OfrenDialsa/go-gin-starter/cmd/api"
	"github/OfrenDialsa/go-gin-starter/cmd/nsq/consumer"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"

	"github.com/rs/zerolog/log"
)

func InitProducer(env *config.EnvironmentVariable) service.ProducerService {
	log.Info().Msg("[>] InitProducer ")

	nsqProducer, err := lib.NsqNewProducer(env.MessageQueue.NSQ.Host)
	if err != nil {
		log.Panic().Msg("producerConf.NewProducer failed")
		panic(err)
	}

	return service.NewProducerServiceImpl(env, nsqProducer)
}

type NsqConsumer struct {
	env             *config.EnvironmentVariable
	Address         string
	ConsumerService service.ConsumerService
	ProducerService service.ProducerService
}

func InitConsumer(env *config.EnvironmentVariable, ps service.ProducerService, setup *api.Setup) error {
	log.Info().Msg("[>] Init Email Consumer")

	cs := service.NewConsumerService(env, setup.Mailer)

	consumer := &NsqConsumer{
		env:             env,
		ConsumerService: cs,
		Address:         env.MessageQueue.NSQ.Host,
		ProducerService: ps,
	}

	err := consumer.ListenSendEmail()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start email listener")
		return err
	}

	return nil
}

func (c *NsqConsumer) ListenSendEmail() error {
	log.Info().
		Str("host", c.Address).
		Msg("[>] Registering Email Worker Listener")

	handler := consumer.NewEmailHandler(c.env, c.ConsumerService)

	if err := handler.StartListen(); err != nil {
		log.Error().Err(err).Msg("EmailHandler.StartListen failed")
		return err
	}

	return nil
}
