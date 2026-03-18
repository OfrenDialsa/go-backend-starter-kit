package repository

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	GetByEmailOrUsername(ctx context.Context, email, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByUserId(ctx context.Context, userId string) (*model.User, error)
	Create(ctx context.Context, tx pgx.Tx, user *model.User) error
	Update(ctx context.Context, tx pgx.Tx, user *model.User) error
	UpdateVerifiedEmail(ctx context.Context, tx pgx.Tx, userId string) error
	UpdateAvatar(ctx context.Context, tx pgx.Tx, userId string, avatarUrl *string) error
	UpdateLastLogin(ctx context.Context, userId string, time time.Time) error
	UpdatePassword(ctx context.Context, userId string, password string) error
	Delete(ctx context.Context, tx pgx.Tx, userId string) error
	CheckUsernameExist(ctx context.Context, username string) (bool, error)
	CheckEmailExist(ctx context.Context, email string) (bool, error)
}
