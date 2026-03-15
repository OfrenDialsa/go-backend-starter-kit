package external

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/external/storage"
	"time"
)

type ExternalService struct {
	Storage storage.StorageService
}

func NewExternalService(env *config.EnvironmentVariable) (*ExternalService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	storageService, err := storage.New(ctx, env)
	if err != nil {
		return nil, err
	}

	return &ExternalService{
		Storage: storageService,
	}, nil
}
