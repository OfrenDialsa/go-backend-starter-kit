package config

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const EnvFolder = "env"
const EnvSecretFilename = ".secret.env"
const EnvFilename = ".env"

func LoadEnv() (env *EnvironmentVariable, err error) {
	envFile := fmt.Sprintf("%s/%s", EnvFolder, EnvFilename)
	// envSecretFile := fmt.Sprintf("%s/%s", EnvFolder, EnvSecretFilename)

	v := viper.New()

	if _, err := os.Stat(envFile); err == nil {
		v.SetConfigFile(envFile)
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Error reading .env file: %v", err)
			panic(err)
		}
		log.Info().Msg(".env file loaded successfully")
	} else {
		v.AutomaticEnv()
		log.Info().Msg(".env file not found, skipping loading")
	}

	err = v.Unmarshal(&env)
	if err != nil {
		log.Error().Err(err).Msg("viper error unmarshal config")
	}

	// Validate Required Value
	err = env.validateRequiredValue()
	if err != nil {
		log.Error().Err(err).Msg("Some required configuration are missing")
		return nil, err
	}

	log.Info().Msg("Env Loaded")
	return
}

type EnvironmentVariable struct {
	App struct {
		Host string `mapstructure:"HOST"`
		Mode string `mapstructure:"MODE"`
	} `mapstructure:"APP"`
	Database struct {
		Timeout  time.Duration `mapstructure:"TIMEOUT"`
		Postgres struct {
			Host     string `mapstructure:"HOST"`
			Port     string `mapstructure:"PORT"`
			Name     string `mapstructure:"NAME"`
			Username string `mapstructure:"USERNAME"`
			Password string `mapstructure:"PASSWORD"`
		} `mapstructure:"POSTGRES"`
		Redis struct {
			IsEnabled bool   `mapstructure:"IS_ENABLED"`
			Host      string `mapstructure:"HOST"`
			Port      int    `mapstructure:"PORT"`
			Password  string `mapstructure:"PASSWORD"`
		} `mapstructure:"REDIS"`
	} `mapstructure:"DATABASE"`
	Swagger struct {
		BasePath    string `mapstructure:"BASE_PATH"`
		Host        string `mapstructure:"HOST"`
		Title       string `mapstructure:"TITLE"`
		Description string `mapstructure:"DESCRIPTION"`
		Version     string `mapstructure:"VERSION"`
	} `mapstructure:"SWAGGER"`
	JWT struct {
		SecretKey struct {
			Access  string `mapstructure:"ACCESS"`
			Refresh string `mapstructure:"REFRESH"`
		} `mapstructure:"SECRET_KEY"`
		Token struct {
			AccessLifeTime time.Duration `mapstructure:"ACCESS_LIFE_TIME"`
		} `mapstructure:"TOKEN"`
	} `mapstructure:"JWT"`
	External struct {
		ResetPasswordURL string `mapstructure:"RESET_PASSWORD_URL"`
		VerifyEmailURL   string `mapstructure:"VERIFY_EMAIL_URL"`
	} `mapstructure:"EXTERNAL"`
	Mail struct {
		From     string `mapstructure:"FROM"`
		FromName string `mapstructure:"FROM_NAME"`
		SMTP     struct {
			Host     string `mapstructure:"HOST"`
			Port     int    `mapstructure:"PORT"`
			User     string `mapstructure:"USER"`
			Password string `mapstructure:"PASSWORD"`
		} `mapstructure:"SMTP"`
	} `mapstructure:"MAIL"`
	Storage struct {
		Type string `mapstructure:"TYPE"`
		S3   struct {
			Endpoint     string `mapstructure:"ENDPOINT"`
			Region       string `mapstructure:"REGION"`
			Bucket       string `mapstructure:"BUCKET"`
			PublicUrl    string `mapstructure:"PUBLIC_URL"`
			AccessKey    string `mapstructure:"ACCESS_KEY"`
			SecretKey    string `mapstructure:"SECRET_KEY"`
			SSL          bool   `mapstructure:"SSL"`
			UsePathStyle bool   `mapstructure:"USE_PATH_STYLE"`
		} `mapstructure:"S3"`
	} `mapstructure:"STORAGE"`
	Prometheus struct {
		Username string `mapstructure:"USERNAME"`
		Password string `mapstructure:"PASSWORD"`
	} `mapstructure:"PROMETHEUS"`
}

func (e *EnvironmentVariable) validateRequiredValue() error {
	return nil
}
