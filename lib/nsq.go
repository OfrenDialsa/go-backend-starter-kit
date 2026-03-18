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
