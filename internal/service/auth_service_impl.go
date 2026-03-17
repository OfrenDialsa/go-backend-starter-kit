package service

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/mailer"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/utils"
	"time"

	"github.com/rs/zerolog/log"
)

type authServiceImpl struct {
	env         *config.EnvironmentVariable
	txStarter   TxStarter
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	Mailer      mailer.Sender
}

func NewAuthService(
	env *config.EnvironmentVariable,
	txStarter TxStarter,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	Mailer mailer.Sender,
) AuthService {
	return &authServiceImpl{
		env:         env,
		txStarter:   txStarter,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		Mailer:      Mailer,
	}
}

func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	existingUser, err := s.userRepo.GetByEmailOrUsername(ctx, req.Email, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if existingUser != nil {
		if existingUser.Email == req.Email {
			return nil, lib.ErrorMessageEmailExists
		}
		if existingUser.Username == req.Username {
			return nil, lib.ErrorMessageUsernameNotAvailable
		}
	}

	hashedPassword, err := lib.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.txStarter.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	userId := utils.Generate()

	user := &model.User{
		UserId:          userId,
		Email:           req.Email,
		Username:        req.Username,
		PasswordHash:    &hashedPassword,
		Name:            req.Name,
		Status:          "active",
		Role:            "user",
		EmailVerifiedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = s.userRepo.Create(ctx, tx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &dto.RegisterResponse{
		User: dto.UserData{
			UserId:    user.UserId,
			Email:     user.Email,
			Username:  user.Username,
			Name:      user.Name,
			Status:    user.Status,
			AvatarURL: user.AvatarURL,
		},
	}, nil
}

func (s *authServiceImpl) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmailOrUsername(ctx, req.Identifier, req.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.PasswordHash == nil {
		return nil, lib.ErrorMessageInvalidCredentials
	}

	if err := lib.Verify(*user.PasswordHash, req.Password); err != nil {
		return nil, lib.ErrorMessageInvalidCredentials
	}

	now := time.Now()
	sessionId := utils.Generate()

	accessExpiry := time.Duration(s.env.JWT.Token.AccessLifeTime)
	refreshExpiry := 7 * 24 * time.Hour

	tokens, err := lib.GenerateTokenPair(
		user.UserId,
		sessionId,
		user.Role,
		s.env.JWT.SecretKey.Access,
		s.env.JWT.SecretKey.Refresh,
		accessExpiry,
		refreshExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	userSession := &model.UserSession{
		SessionId: sessionId,
		UserId:    user.UserId,
		TokenHash: tokens.RefreshToken,
		Type:      "refresh_token",
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		ExpiresAt: now.Add(refreshExpiry),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.sessionRepo.Create(ctx, userSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.UserId, now); err != nil {
		fmt.Printf("failed to update last login: %v\n", err)
	}

	return &dto.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
		User: dto.UserData{
			UserId:    user.UserId,
			Email:     user.Email,
			Username:  user.Username,
			Name:      user.Name,
			Status:    user.Status,
			AvatarURL: user.AvatarURL,
		},
	}, nil
}

func (s *authServiceImpl) Logout(ctx context.Context, sessionID string) error {
	err := s.sessionRepo.RevokeBySessionId(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session during logout: %w", err)
	}
	return nil
}

func (s *authServiceImpl) ForgotPassword(ctx context.Context, email, userAgent, ipAddress string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user by email: %w", err)
	}

	if user == nil {
		return nil
	}

	resetToken, err := utils.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	hashedToken := utils.HashTokenSHA256(resetToken)

	expiresAt := time.Now().Add(time.Hour)

	session := &model.UserSession{
		SessionId: utils.Generate(),
		UserId:    user.UserId,
		TokenHash: hashedToken,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Type:      "reset_password",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.sessionRepo.Create(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create reset password session: %w", err)
	}

	resetPasswordLink := fmt.Sprintf("%s?token=%s", s.env.External.FrontendURL, resetToken)

	emailBody, err := lib.BuildEmailBodyResetPassword(user.Name, resetPasswordLink)
	if err != nil {
		return err
	}

	mailData := dto.MailgunRequest{
		To:          []string{user.Email},
		Subject:     lib.DefaultEmailSubjectResetPassword,
		Body:        emailBody,
		Attachments: []string{},
	}

	_, err = s.Mailer.Send(mailData)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *authServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshTokenResponse, error) {
	claims, err := lib.ValidateToken(refreshToken, s.env.JWT.SecretKey.Refresh)
	if err != nil {
		log.Warn().Err(err).Msg("failed to validate refresh token")
		return nil, fmt.Errorf("invalid refresh token")
	}

	session, err := s.sessionRepo.GetBySessionId(ctx, claims.SessionId)
	if err != nil {
		log.Error().Err(err).Str("session_id", claims.SessionId).Msg("database error when fetching session")
		return nil, fmt.Errorf("failed to process session")
	}
	if session == nil {
		log.Warn().Str("session_id", claims.SessionId).Msg("refresh token session not found")
		return nil, fmt.Errorf("session not found")
	}

	if session.TokenHash != refreshToken {
		log.Error().
			Str("session_id", session.SessionId).
			Str("user_id", claims.UserId).
			Msg("REPLAY ATTACK DETECTED: refresh token already used, revoking session")

		s.sessionRepo.RevokeBySessionId(ctx, session.SessionId)
		return nil, fmt.Errorf("token already used: security breach detected")
	}

	if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		log.Info().Str("session_id", session.SessionId).Msg("attempted to use a revoked or expired session")
		return nil, fmt.Errorf("session revoked or expired")
	}

	accessExpiry := time.Duration(s.env.JWT.Token.AccessLifeTime)
	refreshExpiry := 7 * 24 * time.Hour

	tokens, err := lib.GenerateTokenPair(
		claims.UserId,
		claims.SessionId,
		claims.Role,
		s.env.JWT.SecretKey.Access,
		s.env.JWT.SecretKey.Refresh,
		accessExpiry,
		refreshExpiry,
	)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.UserId).Msg("failed to generate token pair")
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	now := time.Now()
	session.TokenHash = tokens.RefreshToken
	session.UpdatedAt = now
	session.ExpiresAt = now.Add(refreshExpiry)

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		log.Error().Err(err).Str("session_id", session.SessionId).Msg("failed to update session in database")
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *authServiceImpl) ResetPassword(ctx context.Context, token string, newPassword string) error {
	hashedToken := utils.HashTokenSHA256(token)

	log.Debug().Str("token_hash", hashedToken).Msg("attempting to reset password")

	session, err := s.sessionRepo.GetByToken(ctx, hashedToken, "reset_password")
	if err != nil {
		log.Error().Err(err).Str("token_hash", hashedToken).Msg("database error while retrieving reset token")
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		log.Warn().
			Str("token_hash", hashedToken).
			Interface("session_exists", session != nil).
			Msg("invalid or expired reset token attempt")
		return lib.ErrorMessageInvalidResetToken
	}

	hashedPassword, err := lib.Hash(newPassword)
	if err != nil {
		log.Error().Err(err).Str("user_id", session.UserId).Msg("failed to hash new password")
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	err = s.userRepo.UpdatePassword(ctx, session.UserId, hashedPassword)
	if err != nil {
		log.Error().Err(err).Str("user_id", session.UserId).Msg("failed to update password in database")
		return fmt.Errorf("failed to update password: %w", err)
	}

	err = s.sessionRepo.RevokeAllUserSessions(ctx, nil, session.UserId)
	if err != nil {
		log.Warn().Err(err).Str("user_id", session.UserId).Msg("password updated but failed to revoke other sessions")
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	err = s.sessionRepo.DeleteSession(ctx, session.SessionId)
	if err != nil {
		log.Warn().Err(err).Str("session_id", session.SessionId).Msg("failed to delete reset token after use")
	}

	log.Info().
		Str("user_id", session.UserId).
		Str("session_id", session.SessionId).
		Msg("password reset successfully and reset token invalidated")

	return nil
}
