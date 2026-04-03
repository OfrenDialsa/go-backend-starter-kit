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
	"github/OfrenDialsa/go-gin-starter/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type authServiceTestDeps struct {
	userRepo    *mocks.UserRepository
	sessionRepo *mocks.SessionRepository
	logJobRepo  *mocks.LogJobRepository
	txStarter   *mocks.TxStarter
	mockTx      *mocks.Tx
	producerSvc *mocks.ProducerService
	svc         service.AuthService
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func testEnv() *config.EnvironmentVariable {
	env := &config.EnvironmentVariable{}
	env.JWT.SecretKey.Access = "test-access-secret"
	env.JWT.SecretKey.Refresh = "test-refresh-secret"
	env.JWT.Token.AccessLifeTime = 15
	env.MessageQueue.NSQ.Producer.Topic.SendEmail.TopicName = "test-email-topic"
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

func setupAuthService(t *testing.T) *authServiceTestDeps {
	t.Helper()
	d := &authServiceTestDeps{
		userRepo:    mocks.NewUserRepository(t),
		sessionRepo: mocks.NewSessionRepository(t),
		logJobRepo:  mocks.NewLogJobRepository(t),
		txStarter:   mocks.NewTxStarter(t),
		mockTx:      mocks.NewTx(t),
		producerSvc: mocks.NewProducerService(t),
	}

	d.svc = service.NewAuthService(
		testEnv(),
		d.txStarter,
		d.userRepo,
		d.sessionRepo,
		d.logJobRepo,
		d.producerSvc,
	)
	return d
}

func TestRegister(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *dto.RegisterRequest
		setupMock   func(d *authServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name: "Success",
			req: &dto.RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmailOrUsername", ctx, "test@example.com", "testuser").Return(nil, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.userRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.User")).Return(nil)
				d.sessionRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.UserSession")).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.LogJob")).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.MatchedBy(func(p dto.DomainEvent) bool {
					return p.EventType == lib.NSQ_USER_REGISTERED_EVENT
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_EmailAlreadyExists",
			req:  &dto.RegisterRequest{Email: "exists@mail.com", Username: "user1"},
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmailOrUsername", ctx, "exists@mail.com", "user1").
					Return(&model.User{Email: "exists@mail.com"}, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrEmailAlreadyExists,
		},
		{
			name: "Error_DatabaseCheckFailed",
			req:  &dto.RegisterRequest{Email: "new@mail.com", Username: "newuser"},
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmailOrUsername", ctx, mock.Anything, mock.Anything).
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)

			tt.setupMock(d)

			resp, err := d.svc.Register(ctx, "UA", "127.0.0.1", tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.req.Email, resp.User.Email)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
			d.producerSvc.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	ctx := context.Background()
	validToken := "valid-token-123"
	hashedToken := utils.HashTokenSHA256(validToken)
	userId := "user-123"

	tests := []struct {
		name        string
		token       string
		setupMock   func(d *authServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name:  "Success",
			token: validToken,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{
					SessionId: "sess-1",
					UserId:    userId,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				user := &model.User{UserId: userId, Email: "test@example.com", Name: "Test User"}

				d.sessionRepo.On("GetByToken", ctx, hashedToken, "verify_email").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("MarkVerifiedEmail", ctx, d.mockTx, userId).Return(nil)
				d.sessionRepo.On("DeleteSession", ctx, d.mockTx, session.SessionId).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.LogJob")).Return(nil)
				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "Error_TokenExpired",
			token: validToken,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				}
				d.sessionRepo.On("GetByToken", ctx, hashedToken, "verify_email").Return(session, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrInvalidToken,
		},
		{
			name:  "Error_TokenNotFound",
			token: "wrong-token",
			setupMock: func(d *authServiceTestDeps) {
				d.sessionRepo.On("GetByToken", ctx, mock.Anything, "verify_email").Return(nil, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrInvalidToken,
		},
		{
			name:  "Error_UpdateDBFailed_ShouldRollback",
			token: validToken,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{UserId: userId, ExpiresAt: time.Now().Add(1 * time.Hour)}
				user := &model.User{UserId: userId}

				d.sessionRepo.On("GetByToken", ctx, hashedToken, "verify_email").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("MarkVerifiedEmail", ctx, d.mockTx, userId).Return(errors.New("db error"))
				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			err := d.svc.VerifyEmail(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
			d.producerSvc.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestResendVerificationEmail(t *testing.T) {
	ctx := context.Background()
	email := "ofren.dialsa@example.com"
	userAgent := "Mozilla/5.0"
	ipAddress := "127.0.0.1"

	tests := []struct {
		name        string
		req         *dto.ResendVerificationRequest
		setupMock   func(d *authServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name: "Success",
			req:  &dto.ResendVerificationRequest{Email: email},
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{
					UserId:          "user-ulid",
					Email:           email,
					Name:            "Ofren Dialsa",
					EmailVerifiedAt: nil, // Belum verifikasi
				}

				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				// Cleanup token lama
				d.sessionRepo.On("DeleteByType", ctx, d.mockTx, user.UserId, "verify_email").Return(nil)

				// Simpan session baru & log job
				d.sessionRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.UserSession")).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.LogJob")).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				// Kirim ke NSQ
				d.producerSvc.On("PublishEvent", mock.AnythingOfType("dto.DomainEvent")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_UserNotFound",
			req:  &dto.ResendVerificationRequest{Email: "unknown@example.com"},
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmail", ctx, "unknown@example.com").Return(nil, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrUserNotFound,
		},
		{
			name: "Error_AlreadyVerified",
			req:  &dto.ResendVerificationRequest{Email: email},
			setupMock: func(d *authServiceTestDeps) {
				now := time.Now()
				user := &model.User{
					UserId:          "user-ulid",
					Email:           email,
					EmailVerifiedAt: &now, // Sudah verifikasi
				}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrEmailAlreadyVerified,
		},
		{
			name: "Error_TransactionBeginFailed",
			req:  &dto.ResendVerificationRequest{Email: email},
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Email: email}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(nil, errors.New("db connection lost"))
			},
			wantErr: true,
		},
		{
			name: "Success_ButNSQFailed_ShouldMarkJobAsFailed",
			req:  &dto.ResendVerificationRequest{Email: email},
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Email: email}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.sessionRepo.On("DeleteByType", ctx, d.mockTx, mock.Anything, mock.Anything).Return(nil)
				d.sessionRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				// Simulasi NSQ Down
				d.producerSvc.On("PublishEvent", mock.Anything).Return(errors.New("nsq error"))
				// Harus memanggil MarkAsFailed
				d.logJobRepo.On("MarkAsFailed", ctx, nil, mock.Anything, "nsq error").Return(nil)
			},
			wantErr: false, // Return nil karena job sudah di-commit di DB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			err := d.svc.ResendVerificationEmail(ctx, userAgent, ipAddress, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.producerSvc.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	hash := hashPassword(t)

	tests := []struct {
		name        string
		req         dto.LoginRequest
		setupMock   func(d *authServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name: "Success",
			req:  dto.LoginRequest{Identifier: "test@example.com", Password: "password123"},
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{
					UserId:          "user-1",
					Email:           "test@example.com",
					PasswordHash:    &hash,
					Role:            "user",
					EmailVerifiedAt: &now,
					Status:          "active",
				}
				d.userRepo.On("GetByEmailOrUsername", ctx, "test@example.com", "test@example.com").Return(user, nil)
				d.sessionRepo.On("Create", ctx, nil, mock.AnythingOfType("*model.UserSession")).Return(nil)
				d.userRepo.On("MarkLastLogin", ctx, user.UserId, mock.AnythingOfType("time.Time")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_InvalidCredentials_UserNotFound",
			req:  dto.LoginRequest{Identifier: "wrong@example.com", Password: "any"},
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmailOrUsername", ctx, "wrong@example.com", "wrong@example.com").Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "Error_AccountInactive",
			req:  dto.LoginRequest{Identifier: "inactive@example.com", Password: "password123"},
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{
					UserId:          "user-1",
					Email:           "inactive@example.com",
					PasswordHash:    &hash,
					EmailVerifiedAt: &now,
					Status:          "inactive",
				}
				d.userRepo.On("GetByEmailOrUsername", ctx, "inactive@example.com", "inactive@example.com").Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrAccountInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			resp, err := d.svc.Login(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.Equal(t, "Bearer", resp.TokenType)
			}
			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
		})
	}
}

func TestLogout(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionId string
		setupMock func(d *authServiceTestDeps)
		wantErr   bool
	}{
		{
			name:      "Success",
			sessionId: "sess-123",
			setupMock: func(d *authServiceTestDeps) {
				d.sessionRepo.On("RevokeBySessionId", ctx, "sess-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "Error_DatabaseFailure",
			sessionId: "sess-123",
			setupMock: func(d *authServiceTestDeps) {
				d.sessionRepo.On("RevokeBySessionId", ctx, "sess-123").
					Return(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name:      "Error_SessionNotFound",
			sessionId: "ghost-session",
			setupMock: func(d *authServiceTestDeps) {
				d.sessionRepo.On("RevokeBySessionId", ctx, "ghost-session").
					Return(errors.New("session not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			err := d.svc.Logout(ctx, tt.sessionId)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			d.sessionRepo.AssertExpectations(t)
		})
	}
}

func TestForgotPassword(t *testing.T) {
	ctx := context.Background()
	email := "ofren@example.com"
	ua := "Mozilla/5.0"
	ip := "127.0.0.1"

	tests := []struct {
		name      string
		email     string
		setupMock func(d *authServiceTestDeps)
		wantErr   bool
	}{
		{
			name:  "Success",
			email: email,
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Name: "Ofren", Email: email}

				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.sessionRepo.On("Create", ctx, d.mockTx, mock.MatchedBy(func(s *model.UserSession) bool {
					return s.UserId == user.UserId && s.Type == "reset_password"
				})).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.LogJob")).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.MatchedBy(func(p dto.DomainEvent) bool {
					return p.EventType == lib.NSQ_PASSWORD_RESET_REQUESTED_EVENT
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "Success_UserNotFound_ShouldNotReturnError",
			email: "unknown@example.com",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmail", ctx, "unknown@example.com").Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name:  "Error_DatabaseGetByEmailFailed",
			email: email,
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("GetByEmail", ctx, email).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:  "Error_TransactionBeginFailed",
			email: email,
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Email: email}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(nil, errors.New("tx failed"))
			},
			wantErr: true,
		},
		{
			name:  "Error_SessionCreateFailed_ShouldRollback",
			email: email,
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Email: email}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.sessionRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(errors.New("insert failed"))
				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr: true,
		},
		{
			name:  "Success_ProducerFailed_ShouldMarkJobAsFailed",
			email: email,
			setupMock: func(d *authServiceTestDeps) {
				user := &model.User{UserId: "u1", Email: email}
				d.userRepo.On("GetByEmail", ctx, email).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.sessionRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)
				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.Anything).Return(errors.New("nsq down"))
				d.logJobRepo.On("MarkAsFailed", ctx, nil, mock.Anything, "nsq down").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			err := d.svc.ForgotPassword(ctx, tt.email, ua, ip)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
			d.producerSvc.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestRefreshToken(t *testing.T) {
	ctx := context.Background()
	env := testEnv()

	generateTestToken := func(userId, sessId string) string {
		tokenPair, _ := lib.GenerateTokenPair(
			userId, sessId, "user",
			env.JWT.SecretKey.Access, env.JWT.SecretKey.Refresh,
			15*time.Minute, 7*24*time.Hour,
		)
		return tokenPair.RefreshToken
	}

	tests := []struct {
		name         string
		refreshToken string
		setupMock    func(d *authServiceTestDeps, token string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "Success",
			refreshToken: generateTestToken("user-1", "sess-1"),
			setupMock: func(d *authServiceTestDeps, token string) {
				session := &model.UserSession{
					SessionId: "sess-1",
					UserId:    "user-1",
					TokenHash: token,
					ExpiresAt: time.Now().Add(time.Hour),
				}
				d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(session, nil)
				d.sessionRepo.On("Update", ctx, mock.AnythingOfType("*model.UserSession")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Error_ReplayAttack_TokenMismatch",
			refreshToken: generateTestToken("user-1", "sess-1"),
			setupMock: func(d *authServiceTestDeps, token string) {
				session := &model.UserSession{
					SessionId: "sess-1",
					TokenHash: "different-token-hash-in-db",
					ExpiresAt: time.Now().Add(time.Hour),
				}

				d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(session, nil)
				d.sessionRepo.On("RevokeBySessionId", ctx, "sess-1").Return(nil)
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
		{
			name:         "Error_TokenExpired",
			refreshToken: generateTestToken("user-1", "sess-1"),
			setupMock: func(d *authServiceTestDeps, token string) {
				session := &model.UserSession{
					SessionId: "sess-1",
					TokenHash: token,
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				}
				d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(session, nil)
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
		{
			name:         "Error_SessionNotFound",
			refreshToken: generateTestToken("user-1", "sess-1"),
			setupMock: func(d *authServiceTestDeps, token string) {
				d.sessionRepo.On("GetBySessionId", ctx, "sess-1").Return(nil, nil)
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
		{
			name:         "Error_InvalidJWTFormat",
			refreshToken: "not-a-valid-jwt-string",
			setupMock:    func(d *authServiceTestDeps, token string) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d, tt.refreshToken)

			resp, err := d.svc.RefreshToken(ctx, tt.refreshToken)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}

			d.sessionRepo.AssertExpectations(t)
		})
	}
}
func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	validToken := "valid-reset-token-123"
	hashedToken := utils.HashTokenSHA256(validToken)
	userId := "user-123"
	newPassword := "StrongPass123!"

	tests := []struct {
		name        string
		token       string
		password    string
		setupMock   func(d *authServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name:     "Success",
			token:    validToken,
			password: newPassword,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{
					SessionId: "sess-99",
					UserId:    userId,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				user := &model.User{UserId: userId, Email: "test@example.com", Name: "Ofren"}

				d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("UpdatePassword", ctx, d.mockTx, userId, mock.AnythingOfType("string")).Return(nil)
				d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, userId).Return(nil)
				d.sessionRepo.On("DeleteSession", ctx, d.mockTx, "sess-99").Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.AnythingOfType("*model.LogJob")).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "Error_TokenExpired",
			token:    validToken,
			password: newPassword,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				}
				d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrInvalidToken,
		},
		{
			name:     "Error_WeakPassword",
			token:    validToken,
			password: "123",
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{UserId: userId, ExpiresAt: time.Now().Add(1 * time.Hour)}
				user := &model.User{UserId: userId}
				d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrWeakPassword,
		},
		{
			name:     "Error_UpdatePasswordFailed_ShouldRollback",
			token:    validToken,
			password: newPassword,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{UserId: userId, ExpiresAt: time.Now().Add(1 * time.Hour)}
				user := &model.User{UserId: userId}

				d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("UpdatePassword", ctx, d.mockTx, userId, mock.Anything).Return(errors.New("db error"))
				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr: true,
		},
		{
			name:     "Error_ProducerFailed_ShouldMarkJobAsFailed",
			token:    validToken,
			password: newPassword,
			setupMock: func(d *authServiceTestDeps) {
				session := &model.UserSession{SessionId: "s-reset", UserId: userId, ExpiresAt: time.Now().Add(1 * time.Hour)}
				user := &model.User{UserId: userId, Email: "ofren@mail.com"}

				d.sessionRepo.On("GetByToken", ctx, hashedToken, "reset_password").Return(session, nil)
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("UpdatePassword", ctx, d.mockTx, userId, mock.Anything).Return(nil)
				d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, userId).Return(nil)
				d.sessionRepo.On("DeleteSession", ctx, d.mockTx, "s-reset").Return(nil)
				d.logJobRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)
				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

				d.producerSvc.On("PublishEvent", mock.Anything).Return(errors.New("nsq error"))
				d.logJobRepo.On("MarkAsFailed", ctx, nil, mock.Anything, "nsq error").Return(nil)
			},

			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			err := d.svc.ResetPassword(ctx, tt.token, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.logJobRepo.AssertExpectations(t)
			d.producerSvc.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestCheckEmail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		email     string
		setupMock func(d *authServiceTestDeps)
		wantExist bool
		wantErr   bool
	}{
		{
			name:  "Exist",
			email: "test@example.com",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckEmailExist", ctx, "test@example.com").Return(true, nil)
			},
			wantExist: true,
			wantErr:   false,
		},
		{
			name:  "NotExist",
			email: "available@example.com",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckEmailExist", ctx, "available@example.com").Return(false, nil)
			},
			wantExist: false,
			wantErr:   false,
		},
		{
			name:  "Error_DatabaseFailure",
			email: "test@example.com",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckEmailExist", ctx, "test@example.com").
					Return(false, errors.New("db connection failure"))
			},
			wantExist: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			exists, err := d.svc.CheckEmail(ctx, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExist, exists)
			}
			d.userRepo.AssertExpectations(t)
		})
	}
}

func TestCheckUsername(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		username  string
		setupMock func(d *authServiceTestDeps)
		wantExist bool
		wantErr   bool
	}{
		{
			name:     "Exist",
			username: "existinguser",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckUsernameExist", ctx, "existinguser").Return(true, nil)
			},
			wantExist: true,
			wantErr:   false,
		},
		{
			name:     "NotExist",
			username: "newuser",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckUsernameExist", ctx, "newuser").Return(false, nil)
			},
			wantExist: false,
			wantErr:   false,
		},
		{
			name:     "Error_QueryTimeout",
			username: "newuser",
			setupMock: func(d *authServiceTestDeps) {
				d.userRepo.On("CheckUsernameExist", ctx, "newuser").
					Return(false, errors.New("query timeout"))
			},
			wantExist: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupAuthService(t)
			tt.setupMock(d)

			exists, err := d.svc.CheckUsername(ctx, tt.username)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExist, exists)
			}
			d.userRepo.AssertExpectations(t)
		})
	}
}
