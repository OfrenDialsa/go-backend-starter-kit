package storage

import (
	"context"
	"errors"
	"github/OfrenDialsa/go-gin-starter/config"

	awss3 "github/OfrenDialsa/go-gin-starter/external/storage/aws_s3"
	"github/OfrenDialsa/go-gin-starter/external/storage/minio"
)

func New(ctx context.Context, env *config.EnvironmentVariable) (StorageService, error) {
	switch env.Storage.Type {
	case "minio":
		return minio.NewMinioStorageManager(env)
	case "s3":
		return awss3.NewS3StorageManager(ctx, env)
	default:
		return nil, errors.New("unsupported storage driver")
	}
}
