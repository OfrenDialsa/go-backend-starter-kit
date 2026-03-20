package repository

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	Create(ctx context.Context, tx pgx.Tx, user *model.User) error
	GetByEmailOrUsername(ctx context.Context, email, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByUserId(ctx context.Context, userId string) (*model.User, error)
	Update(ctx context.Context, tx pgx.Tx, user *model.User) error
	UpdateAvatar(ctx context.Context, tx pgx.Tx, userId string, avatarUrl *string) error
	UpdatePassword(ctx context.Context, tx pgx.Tx, userId string, password string) error
	MarkVerifiedEmail(ctx context.Context, tx pgx.Tx, userId string) error
	MarkLastLogin(ctx context.Context, userId string, time time.Time) error
	Delete(ctx context.Context, tx pgx.Tx, userId string) error
	CheckUsernameExist(ctx context.Context, username string) (bool, error)
	CheckEmailExist(ctx context.Context, email string) (bool, error)
}
