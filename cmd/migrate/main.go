package main

import (
	"flag"
	"fmt"

	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"

	"github.com/rs/zerolog/log"
)

func main() {
	var action string
	flag.StringVar(&action, "action", "up", "Migration action: up or down")
	flag.Parse()

	env, err := config.LoadEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env.Database.Postgres.Username,
		env.Database.Postgres.Password,
		env.Database.Postgres.Host,
		env.Database.Postgres.Port,
		env.Database.Postgres.Name,
	)

	switch action {
	case "up":
		log.Info().Msg("Running migrations...")
		if err := database.RunMigrations(databaseURL); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
		log.Info().Msg("Migrations completed successfully")
	case "down":
		log.Info().Msg("Rolling back migrations...")
		if err := database.RollbackMigrations(databaseURL); err != nil {
			log.Fatal().Err(err).Msg("failed to rollback migrations")
		}
		log.Info().Msg("Rollback completed successfully")
	default:
		log.Fatal().Str("action", action).Msg("invalid action. Use 'up' or 'down'")
	}
}
