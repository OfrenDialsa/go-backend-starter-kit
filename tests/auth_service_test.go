package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ===================== Helpers =====================

func testEnv() *config.EnvironmentVariable {
	env := &config.EnvironmentVariable{}
	env.JWT.SecretKey.Access = "test-access-secret"
	env.JWT.SecretKey.Refresh = "test-refresh-secret"
	env.JWT.Token.AccessLifeTime = 15
	return env
}

func hashPassword(t *testing.T) string {
	t.Helper()
	hash, err := lib.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

type authTestDeps struct {
	userRepo    *mocks.UserRepository
	sessionRepo *mocks.SessionRepository
	txStarter   *mocks.TxStarter
	mockTx      *mocks.Tx
	mailer      *mocks.MockMailer
	svc         service.AuthService
}

func setupAuthService(t *testing.T) *authTestDeps {
	t.Helper()
	d := &authTestDeps{
		userRepo:    mocks.NewUserRepository(t),
		sessionRepo: mocks.NewSessionRepository(t),
		txStarter:   mocks.NewTxStarter(t),
		mockTx:      mocks.NewTx(t),
		mailer:      mocks.NewMockMailer(t),
	}

	d.svc = service.NewAuthService(
		testEnv(),
		d.txStarter,
		d.userRepo,
		d.sessionRepo,
		d.mailer,
	)
	return d
}

// ===================== Register Tests =====================

func TestRegister_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	req := &dto.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
		Name:     "Test User",
	}

	d.userRepo.On("GetByEmailOrUsername", ctx, req.Email, req.Username).Return(nil, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
	d.userRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.User")).Return(nil)
	d.mockTx.On("Commit", ctx).Return(nil)
	d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

	resp, err := d.svc.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Email, resp.User.Email)
	assert.Equal(t, "active", resp.User.Status)
}

func TestRegister_EmailOrUsernameAlreadyExists(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	existingUser := &model.User{
		UserId:   "existing-123",
		Email:    "test@example.com",
		Username: "testuser",
	}

	d.userRepo.On("GetByEmailOrUsername", ctx, "test@example.com", "testuser").
		Return(existingUser, nil)

	req := &dto.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	resp, err := d.svc.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), lib.ErrEmailAlreadyExists)

	d.userRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestRegister_DatabaseErrorOnCheck(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("GetByEmailOrUsername", ctx, mock.Anything, mock.Anything).
		Return(nil, errors.New("database connection error"))

	req := &dto.RegisterRequest{
		Email:    "new@example.com",
		Username: "newuser",
	}

	resp, err := d.svc.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database connection error")
}

// ===================== Login Tests =====================

func TestLogin_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{
		UserId:       "user-1",
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: &hash,
		Role:         "user",
	}

	d.userRepo.On("GetByEmailOrUsername", ctx, "test@example.com", "test@example.com").Return(user, nil)
	d.sessionRepo.On("Create", ctx, mock.AnythingOfType("*model.UserSession")).Return(nil)
	d.userRepo.On("UpdateLastLogin", ctx, user.UserId, mock.AnythingOfType("time.Time")).Return(nil)

	resp, err := d.svc.Login(ctx, dto.LoginRequest{
		Identifier: "test@example.com",
		Password:   "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("GetByEmailOrUsername", ctx, "wrong@example.com", "wrong@example.com").Return(nil, nil)

	resp, err := d.svc.Login(ctx, dto.LoginRequest{
		Identifier: "wrong@example.com",
		Password:   "any",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ===================== Logout Tests =====================

func TestLogout_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.sessionRepo.On("RevokeBySessionId", ctx, "sess-123").Return(nil)

	err := d.svc.Logout(ctx, "sess-123")
	assert.NoError(t, err)
}

func TestLogout_DatabaseError(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	sessionId := "sess-123"

	d.sessionRepo.On("RevokeBySessionId", ctx, sessionId).
		Return(errors.New("failed to delete session: connection refused"))

	err := d.svc.Logout(ctx, sessionId)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestLogout_SessionNotFound(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	sessionId := "non-existent-session"
	d.sessionRepo.On("RevokeBySessionId", ctx, sessionId).
		Return(errors.New("session not found"))

	err := d.svc.Logout(ctx, sessionId)
	assert.Error(t, err)
}

// ===================== Forgot Password Tests =====================

func TestForgotPassword_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	email := "user@example.com"

	user := &model.User{UserId: "user-1", Name: "Ofren", Email: email}

	d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
	d.sessionRepo.On("Create", ctx, mock.MatchedBy(func(s *model.UserSession) bool {
		return s.UserId == user.UserId && s.Type == "reset_password"
	})).Return(nil)
	d.mailer.On("Send", mock.AnythingOfType("dto.MailgunRequest")).Return("msg-id", nil)

	err := d.svc.ForgotPassword(ctx, email, "Mozilla", "127.0.0.1")

	assert.NoError(t, err)
}

func TestForgotPassword_UserNotFound(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("GetByEmail", ctx, "unknown@example.com").Return(nil, nil)

	err := d.svc.ForgotPassword(ctx, "unknown@example.com", "UA", "IP")

	assert.NoError(t, err)
}

func TestForgotPassword_MailerError(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	email := "ofren@example.com"
	user := &model.User{UserId: "u1", Name: "Ofren", Email: email}

	d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)

	d.sessionRepo.On("Create", ctx, mock.MatchedBy(func(s *model.UserSession) bool {
		return s.Type == "reset_password"
	})).Return(nil)

	d.mailer.On("Send", mock.Anything).Return("", errors.New("mail server connection timeout"))

	err := d.svc.ForgotPassword(ctx, email, "UA", "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestForgotPassword_DatabaseError(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	email := "ofren@example.com"

	d.userRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db connection lost"))

	err := d.svc.ForgotPassword(ctx, email, "UA", "127.0.0.1")

	assert.Error(t, err)
}

func TestForgotPassword_SaveTokenError(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	email := "ofren@example.com"
	user := &model.User{UserId: "u1", Email: email}

	d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)

	d.sessionRepo.On("Create", ctx, mock.Anything).Return(errors.New("failed to persist token"))

	err := d.svc.ForgotPassword(ctx, email, "UA", "127.0.0.1")

	assert.Error(t, err)
}

// ===================== Refresh Token Tests =====================

func TestRefreshToken_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	env := testEnv()
	tokenPair, _ := lib.GenerateTokenPair("user-1", "sess-1", "user", env.JWT.SecretKey.Access, env.JWT.SecretKey.Refresh, 15*time.Minute, 7*24*time.Hour)

	session := &model.UserSession{
		SessionId: "sess-1",
		UserId:    "user-1",
		TokenHash: tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(session, nil)
	d.sessionRepo.On("Update", ctx, mock.AnythingOfType("*model.UserSession")).Return(nil)

	resp, err := d.svc.RefreshToken(ctx, tokenPair.RefreshToken)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestRefreshToken_ReplayAttack(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	env := testEnv()

	accessExpiry := 15 * time.Minute
	refreshExpiry := 7 * 24 * time.Hour

	tokenPair, err := lib.GenerateTokenPair(
		"user-1",
		"sess-1",
		"user",
		env.JWT.SecretKey.Access,
		env.JWT.SecretKey.Refresh,
		accessExpiry,
		refreshExpiry,
	)
	assert.NoError(t, err)

	session := &model.UserSession{
		SessionId: "sess-1",
		TokenHash: "token-lama-yang-sudah-tidak-valid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(session, nil).Once()
	d.sessionRepo.On("RevokeBySessionId", ctx, "sess-1").Return(nil).Once()

	resp, err := d.svc.RefreshToken(ctx, tokenPair.RefreshToken)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security breach detected")
	assert.Nil(t, resp)
}

func TestRefreshToken_Expired(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	env := testEnv()

	tokenPair, _ := lib.GenerateTokenPair("u1", "s1", "user",
		env.JWT.SecretKey.Access, env.JWT.SecretKey.Refresh, 15*time.Minute, 7*24*time.Hour)

	session := &model.UserSession{
		SessionId: "s1",
		TokenHash: tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	d.sessionRepo.On("GetBySessionId", ctx, "s1").Return(session, nil)

	resp, err := d.svc.RefreshToken(ctx, tokenPair.RefreshToken)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session revoked")
	assert.Nil(t, resp)
}

func TestRefreshToken_SessionNotFound(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	env := testEnv()

	tokenPair, _ := lib.GenerateTokenPair("u1", "s1", "user",
		env.JWT.SecretKey.Access, env.JWT.SecretKey.Refresh, 15*time.Minute, 7*24*time.Hour)

	d.sessionRepo.On("GetBySessionId", ctx, "s1").Return(nil, nil)

	resp, err := d.svc.RefreshToken(ctx, tokenPair.RefreshToken)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	assert.Nil(t, resp)
}

func TestRefreshToken_InvalidFormat(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	invalidToken := "bukan-token-jwt-yang-benar"

	resp, err := d.svc.RefreshToken(ctx, invalidToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ===================== Reset Password Tests =====================

func TestResetPassword_Success(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	token := "raw-reset-token"

	session := &model.UserSession{
		SessionId: "sess-reset",
		UserId:    "user-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	d.sessionRepo.On("GetByToken", ctx, mock.Anything, "reset_password").Return(session, nil)
	d.userRepo.On("UpdatePassword", ctx, "user-1", mock.Anything).Return(nil)
	d.sessionRepo.On("RevokeAllUserSessions", ctx, nil, "user-1").Return(nil)
	d.sessionRepo.On("DeleteSession", ctx, "sess-reset").Return(nil)

	err := d.svc.ResetPassword(ctx, token, "newpassword123")

	assert.NoError(t, err)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	session := &model.UserSession{
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	d.sessionRepo.On("GetByToken", ctx, mock.Anything, "reset_password").Return(session, nil)

	err := d.svc.ResetPassword(ctx, "token", "pass")

	assert.Error(t, err)
	assert.Equal(t, lib.ErrorMessageInvalidResetToken, err)
}

func TestResetPassword_TokenNotFound(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	rawToken := "invalid-token"
	hashedToken := hashToken(rawToken)

	d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(nil, nil)

	err := d.svc.ResetPassword(ctx, rawToken, "newpassword123")

	assert.Error(t, err)
	assert.Equal(t, lib.ErrorMessageInvalidResetToken, err)
}

func TestResetPassword_UpdatePasswordError(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	rawToken := "valid-token"
	hashedToken := hashToken(rawToken)
	userId := "user-1"

	session := &model.UserSession{
		SessionId: "sess-id",
		UserId:    userId,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
	d.userRepo.On("UpdatePassword", ctx, userId, mock.Anything).Return(errors.New("db error"))

	err := d.svc.ResetPassword(ctx, rawToken, "newpassword123")

	assert.Error(t, err)
}

func TestResetPassword_RevokeSessionsFailure(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	rawToken := "valid-token"
	hashedToken := hashToken(rawToken)
	userId := "user-1"

	session := &model.UserSession{
		SessionId: "sess-id",
		UserId:    userId,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
	d.userRepo.On("UpdatePassword", ctx, userId, mock.Anything).Return(nil)
	d.sessionRepo.On("RevokeAllUserSessions", ctx, nil, userId).Return(errors.New("failed to revoke"))

	err := d.svc.ResetPassword(ctx, rawToken, "newpassword123")

	assert.Error(t, err)
}

func TestResetPassword_DeleteTokenFailure(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()
	rawToken := "valid-token"
	hashedToken := hashToken(rawToken)
	userId := "user-1"

	session := &model.UserSession{
		SessionId: "sess-id",
		UserId:    userId,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
	d.userRepo.On("UpdatePassword", ctx, userId, mock.Anything).Return(nil)
	d.sessionRepo.On("RevokeAllUserSessions", ctx, nil, userId).Return(nil)
	d.sessionRepo.On("DeleteSession", ctx, "sess-id").Return(errors.New("clean up error"))

	err := d.svc.ResetPassword(ctx, rawToken, "newpassword123")

	assert.Error(t, err)
}

// ===================== Check Existence Tests =====================

func TestCheckEmail_Exist(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckEmailExist", ctx, "test@example.com").Return(true, nil)

	exists, err := d.svc.CheckEmail(ctx, "test@example.com")

	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckUsername_NotExist(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckUsernameExist", ctx, "newuser").Return(false, nil)

	exists, err := d.svc.CheckUsername(ctx, "newuser")

	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckEmail_NotExist(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckEmailExist", ctx, "available@example.com").Return(false, nil)

	exists, err := d.svc.CheckEmail(ctx, "available@example.com")

	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckEmail_Error(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckEmailExist", ctx, "test@example.com").Return(false, errors.New("db connection failure"))

	exists, err := d.svc.CheckEmail(ctx, "test@example.com")

	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "db connection failure")
}

func TestCheckUsername_Exist(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckUsernameExist", ctx, "existinguser").Return(true, nil)

	exists, err := d.svc.CheckUsername(ctx, "existinguser")

	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckUsername_Error(t *testing.T) {
	d := setupAuthService(t)
	ctx := context.Background()

	d.userRepo.On("CheckUsernameExist", ctx, "newuser").Return(false, errors.New("query timeout"))

	exists, err := d.svc.CheckUsername(ctx, "newuser")

	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "query timeout")
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
