package database

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database/postgres"
	"github/OfrenDialsa/go-gin-starter/database/redis"
)

type WrapDB struct {
	Database *postgres.WrapDB
	Redis    *redis.WrapDB
}

func InitDB(env *config.EnvironmentVariable) *WrapDB {
	database := postgres.InitDatabase(env)
	redis := redis.InitDatabase(env)

	return &WrapDB{
		Database: database,
		Redis:    redis,
	}
}
