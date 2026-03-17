package redis

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// init redis
type WrapDB struct {
	Conn *redis.Client
}

func InitDatabase(env *config.EnvironmentVariable) *WrapDB {
	host := fmt.Sprintf(`%s:%d`, env.Database.Redis.Host, env.Database.Redis.Port)
	password := env.Database.Redis.Password

	var client *redis.Client
	if env.Database.Redis.IsEnabled {

		client = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       0,
		})

		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Panic().Err(err).Msg("failed to connect redis")
		}
	}

	return &WrapDB{
		Conn: client,
	}
}
