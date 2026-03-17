package service

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
)

type UserService interface {
	GetMe(ctx context.Context, userId string) (*dto.GetMeResponse, error)
	ChangePassword(ctx context.Context, req dto.ChangePasswordRequest) error
	UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (*dto.UpdateProfileResponse, error)
	DeleteAccount(ctx context.Context, req dto.UserDeleteAccountRequest) error
	DeleteAvatar(ctx context.Context, userId string) error
}
