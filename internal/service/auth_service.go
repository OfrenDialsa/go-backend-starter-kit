package service

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
)

type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshTokenResponse, error)
	Logout(ctx context.Context, sessionId string) error
	ForgotPassword(ctx context.Context, email, userAgent, ipAddress string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
}
