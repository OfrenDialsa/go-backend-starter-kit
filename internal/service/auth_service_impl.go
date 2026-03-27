package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/metrics"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/utils"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type authServiceImpl struct {
	env         *config.EnvironmentVariable
	txStarter   TxStarter
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	logJobRepo  repository.LogJobRepository
	producerSvc ProducerService
}

func NewAuthService(
	env *config.EnvironmentVariable,
	txStarter TxStarter,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	logJobRepo repository.LogJobRepository,
	producerSvc ProducerService,
) AuthService {
	return &authServiceImpl{
		env:         env,
		txStarter:   txStarter,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		logJobRepo:  logJobRepo,
		producerSvc: producerSvc,
	}
}

func (s *authServiceImpl) Register(ctx context.Context, userAgent, ipAddress string, req *dto.RegisterRequest) (res *dto.RegisterResponse, err error) {
	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		metrics.TrackAuth("login", status, time.Since(start))
	}()

	existingUser, err := s.userRepo.GetByEmailOrUsername(ctx, req.Email, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if existingUser != nil {
		if existingUser.Email == req.Email {
			return nil, lib.ErrEmailAlreadyExists
		}
		if existingUser.Username == req.Username {
			return nil, lib.ErrUsernameNotAvailable
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
	userId := utils.GenerateULID()
	username := strings.ToLower(req.Username)

	user := &model.User{
		UserId:          userId,
		Email:           req.Email,
		Username:        username,
		PasswordHash:    &hashedPassword,
		Name:            req.Name,
		Status:          "active",
		Role:            "user",
		EmailVerifiedAt: nil,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = s.userRepo.Create(ctx, tx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	verifToken, err := utils.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	hashedToken := utils.HashTokenSHA256(verifToken)

	expiresAt := time.Now().Add(time.Hour)

	session := &model.UserSession{
		SessionId: utils.GenerateULID(),
		UserId:    user.UserId,
		TokenHash: hashedToken,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Type:      "verify_email",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.sessionRepo.Create(ctx, tx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create reset password session: %w", err)
	}

	verifLink := fmt.Sprintf("%s?token=%s", s.env.External.VerifyEmailURL, verifToken)

	jobId := utils.GenerateULID()

	mailPayload := dto.EmailTaskPayload{
		JobId: jobId,
		Type:  "verify_email",
		Email: user.Email,
		Name:  user.Name,
		Link:  verifLink,
	}

	payloadBytes, err := json.Marshal(mailPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %w", err)
	}

	job := &model.LogJob{
		JobId:       mailPayload.JobId,
		Type:        mailPayload.Type,
		Payload:     payloadBytes,
		Status:      "pending",
		RetryCount:  0,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err = s.logJobRepo.Create(ctx, tx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create log job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = s.producerSvc.SendEmailRequest(mailPayload)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish email to NSQ")
		s.logJobRepo.MarkAsFailed(ctx, nil, job.JobId, err.Error())
		return nil, fmt.Errorf("failed to queue email: %w", err)
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

func (s *authServiceImpl) ResendVerificationEmail(ctx context.Context, userAgent, ipAddress string, req *dto.ResendVerificationRequest) (err error) {
	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		metrics.TrackAuth("resend_verification", status, time.Since(start))
	}()

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return lib.ErrUserNotFound
	}

	if user.EmailVerifiedAt != nil {
		return lib.ErrEmailAlreadyVerified
	}

	tx, err := s.txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = s.sessionRepo.DeleteByType(ctx, tx, user.UserId, "verify_email")
	if err != nil {
		return fmt.Errorf("failed to cleanup old verification tokens: %w", err)
	}

	verifToken, err := utils.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	hashedToken := utils.HashTokenSHA256(verifToken)
	expiresAt := time.Now().Add(time.Hour)

	session := &model.UserSession{
		SessionId: utils.GenerateULID(),
		UserId:    user.UserId,
		TokenHash: hashedToken,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Type:      "verify_email",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.sessionRepo.Create(ctx, tx, session)
	if err != nil {
		return fmt.Errorf("failed to create verification session: %w", err)
	}

	verifLink := fmt.Sprintf("%s?token=%s", s.env.External.VerifyEmailURL, verifToken)
	jobId := utils.GenerateULID()

	mailPayload := dto.EmailTaskPayload{
		JobId: jobId,
		Type:  "verify_email",
		Email: user.Email,
		Name:  user.Name,
		Link:  verifLink,
	}

	payloadBytes, _ := json.Marshal(mailPayload)
	job := &model.LogJob{
		JobId:       jobId,
		Type:        mailPayload.Type,
		Payload:     payloadBytes,
		Status:      "pending",
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err = s.logJobRepo.Create(ctx, tx, job)
	if err != nil {
		return fmt.Errorf("failed to create log job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = s.producerSvc.SendEmailRequest(mailPayload)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish resend email to NSQ")
		s.logJobRepo.MarkAsFailed(ctx, nil, jobId, err.Error())
	}

	return nil
}

func (s *authServiceImpl) VerifyEmail(ctx context.Context, token string) error {
	hashedToken := utils.HashTokenSHA256(token)

	session, err := s.sessionRepo.GetByToken(ctx, hashedToken, "verify_email")
	if err != nil {
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return lib.ErrInvalidToken
	}

	user, err := s.userRepo.GetByUserId(ctx, session.UserId)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	tx, err := s.txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = s.userRepo.MarkVerifiedEmail(ctx, tx, session.UserId)
	if err != nil {
		return fmt.Errorf("failed to update verified email: %w", err)
	}

	err = s.sessionRepo.DeleteSession(ctx, tx, session.SessionId)
	if err != nil {
		return fmt.Errorf("failed to cleanup session: %w", err)
	}

	loginLink := s.env.External.FrontendURL + "/login"
	jobId := utils.GenerateULID()
	mailPayload := dto.EmailTaskPayload{
		JobId: jobId,
		Type:  "verify_email_success",
		Email: user.Email,
		Name:  user.Name,
		Link:  loginLink,
	}

	payloadBytes, err := json.Marshal(mailPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	job := &model.LogJob{
		JobId:       mailPayload.JobId,
		Type:        mailPayload.Type,
		Payload:     payloadBytes,
		Status:      "pending",
		RetryCount:  0,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err = s.logJobRepo.Create(ctx, tx, job)
	if err != nil {
		return fmt.Errorf("failed to create log job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = s.producerSvc.SendEmailRequest(mailPayload)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish email to NSQ")
		s.logJobRepo.MarkAsFailed(ctx, nil, job.JobId, err.Error())

		return fmt.Errorf("failed to queue email: %w", err)
	}

	return nil
}

func (s *authServiceImpl) Login(ctx context.Context, req dto.LoginRequest) (res *dto.LoginResponse, err error) {
	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		metrics.TrackAuth("login", status, time.Since(start))
	}()

	user, err := s.userRepo.GetByEmailOrUsername(ctx, req.Identifier, req.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.PasswordHash == nil {
		return nil, lib.ErrInvalidCredential
	}

	if user.EmailVerifiedAt == nil {
		return nil, lib.ErrEmailNotVerified
	}

	if user.Status != "active" {
		return nil, lib.ErrAccountInactive
	}

	if err := lib.Verify(*user.PasswordHash, req.Password); err != nil {
		return nil, lib.ErrInvalidCredential
	}

	now := time.Now()
	sessionId := utils.GenerateULID()

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

	if err := s.sessionRepo.Create(ctx, nil, userSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := s.userRepo.MarkLastLogin(ctx, user.UserId, now); err != nil {
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
		SessionId: utils.GenerateULID(),
		UserId:    user.UserId,
		TokenHash: hashedToken,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Type:      "reset_password",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resetPasswordLink := fmt.Sprintf("%s?token=%s", s.env.External.ResetPasswordURL, resetToken)

	jobId := utils.GenerateULID()
	mailPayload := dto.EmailTaskPayload{
		JobId: jobId,
		Type:  "forgot_password",
		Email: user.Email,
		Name:  user.Name,
		Link:  resetPasswordLink,
	}

	payloadBytes, err := json.Marshal(mailPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	job := &model.LogJob{
		JobId:       mailPayload.JobId,
		Type:        mailPayload.Type,
		Payload:     payloadBytes,
		Status:      "pending",
		RetryCount:  0,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	tx, err := s.txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = s.sessionRepo.Create(ctx, tx, session)
	if err != nil {
		return fmt.Errorf("failed to create reset password session: %w", err)
	}

	err = s.logJobRepo.Create(ctx, tx, job)
	if err != nil {
		return fmt.Errorf("failed to create log job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = s.producerSvc.SendEmailRequest(mailPayload)
	if err != nil {
		log.Error().Err(err).
			Str("job_id", job.JobId).
			Msg("failed to publish forgot password email to NSQ")

		s.logJobRepo.MarkAsFailed(ctx, nil, job.JobId, err.Error())
	}

	return nil
}

func (s *authServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (res *dto.RefreshTokenResponse, err error) {
	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		metrics.TrackAuth("refresh_token", status, time.Since(start))
	}()

	claims, err := lib.ValidateToken(refreshToken, s.env.JWT.SecretKey.Refresh)
	if err != nil {
		log.Warn().Err(err).Msg("failed to validate refresh token")
		return nil, lib.ErrInvalidToken
	}

	session, err := s.sessionRepo.GetBySessionId(ctx, claims.SessionId)
	if err != nil {
		log.Error().Err(err).Str("session_id", claims.SessionId).Msg("database error when fetching session")
		return nil, fmt.Errorf("failed to process session")
	}
	if session == nil {
		log.Warn().Str("session_id", claims.SessionId).Msg("refresh token session not found")
		return nil, lib.ErrUnauthorized
	}

	if session.TokenHash != refreshToken {
		log.Error().
			Str("session_id", session.SessionId).
			Str("user_id", claims.UserId).
			Msg("REPLAY ATTACK DETECTED: refresh token already used, revoking session")

		s.sessionRepo.RevokeBySessionId(ctx, session.SessionId)
		return nil, lib.ErrUnauthorized
	}

	if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		log.Info().Str("session_id", session.SessionId).Msg("attempted to use a revoked or expired session")
		return nil, lib.ErrUnauthorized
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
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return lib.ErrInvalidToken
	}

	user, err := s.userRepo.GetByUserId(ctx, session.UserId)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	if len(newPassword) < 8 {
		return lib.ErrWeakPassword
	}

	hashedPassword, err := lib.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	jobId := utils.GenerateULID()
	mailPayload := dto.EmailTaskPayload{
		JobId: jobId,
		Type:  "password_reset_success",
		Email: user.Email,
		Name:  user.Name,
	}

	payloadBytes, err := json.Marshal(mailPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	job := &model.LogJob{
		JobId:       mailPayload.JobId,
		Type:        mailPayload.Type,
		Payload:     payloadBytes,
		Status:      "pending",
		RetryCount:  0,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	tx, err := s.txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = s.userRepo.UpdatePassword(ctx, tx, session.UserId, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	err = s.sessionRepo.RevokeAllUserSessions(ctx, tx, session.UserId)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	err = s.sessionRepo.DeleteSession(ctx, tx, session.SessionId)
	if err != nil {
		return fmt.Errorf("failed to delete reset session: %w", err)
	}

	err = s.logJobRepo.Create(ctx, tx, job)
	if err != nil {
		return fmt.Errorf("failed to create log job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = s.producerSvc.SendEmailRequest(mailPayload)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish email to NSQ")
		s.logJobRepo.MarkAsFailed(ctx, nil, job.JobId, err.Error())
	}

	log.Info().
		Str("user_id", session.UserId).
		Msg("password reset successfully and sessions cleaned up")

	return nil
}

func (s *authServiceImpl) CheckEmail(ctx context.Context, email string) (bool, error) {
	exists, err := s.userRepo.CheckEmailExist(ctx, email)
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("failed to check email availability")
		return false, fmt.Errorf("failed to check email: %w", err)
	}
	return exists, nil
}

func (s *authServiceImpl) CheckUsername(ctx context.Context, username string) (bool, error) {
	exists, err := s.userRepo.CheckUsernameExist(ctx, username)
	if err != nil {
		log.Error().Err(err).Str("username", username).Msg("failed to check username availability")
		return false, fmt.Errorf("failed to check username: %w", err)
	}
	return exists, nil
}
