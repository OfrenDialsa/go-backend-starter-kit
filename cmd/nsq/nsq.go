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
	Repositories    api.Repositories
	ConsumerService service.ConsumerService
	ProducerService service.ProducerService
}

func InitConsumer(env *config.EnvironmentVariable, ps service.ProducerService, setup *api.Setup) error {
	log.Info().Msg("[>] Init Consumer")

	cs := service.NewConsumerService(env, setup.Repository.LogJob, setup.Mailer)

	consumer := &NsqConsumer{
		env:             env,
		ConsumerService: cs,
		Address:         env.MessageQueue.NSQ.Host,
		ProducerService: ps,
		Repositories:    setup.Repository,
	}

	err := consumer.ListenEmailConsumer()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start Email Consumer listener")
		return err
	}

	return nil
}

func (c *NsqConsumer) ListenEmailConsumer() error {
	log.Info().
		Str("host", c.Address).
		Msg("[>] Registering Email Consumer Listener")

	handler := consumer.NewEmailConsumer(c.env, c.ConsumerService, c.Repositories)

	if err := handler.StartListen(); err != nil {
		log.Error().Err(err).Msg("ProcessEmailHandler.StartListen failed")
		return err
	}

	return nil
}
