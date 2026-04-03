package lib

import (
	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
)

type ConsumerConfig struct {
	Address string
	Topic   string
	Channel string
}

func NsqNewProducer(addr string) (producer *nsq.Producer, err error) {
	config := nsq.NewConfig()
	producer, err = nsq.NewProducer(addr, config)
	if err != nil {
		log.Error().Err(err).Str("addr", addr).Msg("nsq.NewProducer failed")
		return
	}

	err = producer.Ping()
	if err != nil {
		log.Error().Err(err).Msg("producer.Ping failed")
		return
	}

	return
}

var (
	NSQ_USER_REGISTERED_EVENT          = "USER_REGISTERED"
	NSQ_RESEND_VERIFICATION_EVENT      = "RESEND_VERIFICATION"
	NSQ_PASSWORD_RESET_REQUESTED_EVENT = "PASSWORD_RESET_REQUESTED"
	NSQ_PASSWORD_RESET_SUCCESS_EVENT   = "PASSWORD_RESET_SUCCESS"
	NSQ_EMAIL_VERIFIED_EVENT           = "EMAIL_VERIFIED"
)
