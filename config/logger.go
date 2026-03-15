package config

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const AppModeDev = "dev"
const AppModePreview = "pre"
const AppModeProduction = "prod"

func InitLogger(env *EnvironmentVariable) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if env.App.Mode == AppModePreview {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if env.App.Mode == AppModeProduction {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
}
