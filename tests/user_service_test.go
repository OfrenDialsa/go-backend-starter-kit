package tests

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type userServiceTestDeps struct {
	userRepo    *mocks.UserRepository
	sessionRepo *mocks.SessionRepository
	auditRepo   *mocks.AuditLogRepository
	txStarter   *mocks.TxStarter
	mockTx      *mocks.Tx
	storage     *mocks.StorageService
	svc         service.UserService
}

func setupUserService(t *testing.T) *userServiceTestDeps {
	t.Helper()
	d := &userServiceTestDeps{
		userRepo:    mocks.NewUserRepository(t),
		sessionRepo: mocks.NewSessionRepository(t),
		auditRepo:   mocks.NewAuditLogRepository(t),
		txStarter:   mocks.NewTxStarter(t),
		mockTx:      mocks.NewTx(t),
		storage:     mocks.NewStorageService(t),
	}

	d.svc = service.NewUserService(
		testEnv(),
		d.txStarter,
		d.userRepo,
		d.sessionRepo,
		d.auditRepo,
		d.storage,
	)
	return d
}

func TestGetMe(t *testing.T) {
	ctx := context.Background()
	userId := "user-123"

	tests := []struct {
		name        string
		userId      string
		setupMock   func(d *userServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name:   "Success",
			userId: userId,
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{
					UserId: userId,
					Email:  "ofren@example.com",
					Name:   "Ofren Dialsa",
				}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr: false,
		},
		{
			name:   "Error_UserNotFound",
			userId: "ghost-id",
			setupMock: func(d *userServiceTestDeps) {
				d.userRepo.On("GetByUserId", ctx, "ghost-id").Return(nil, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrUserNotFound,
		},
		{
			name:   "Error_DatabaseFailure",
			userId: userId,
			setupMock: func(d *userServiceTestDeps) {
				d.userRepo.On("GetByUserId", ctx, userId).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupUserService(t)
			tt.setupMock(d)

			resp, err := d.svc.GetMe(ctx, tt.userId)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.userId, resp.User.UserId)
				assert.Equal(t, "Ofren Dialsa", resp.User.Name)
			}

			d.userRepo.AssertExpectations(t)
		})
	}
}
func TestUpdateProfile(t *testing.T) {
	ctx := context.Background()
	userId := "user-123"

	createMockFile := func(filename string) *multipart.FileHeader {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("avatar", filename)
		_, _ = part.Write([]byte("fake-image-content"))
		writer.Close()
		reader := multipart.NewReader(body, writer.Boundary())
		form, _ := reader.ReadForm(1024)
		return form.File["avatar"][0]
	}

	tests := []struct {
		name        string
		req         dto.UpdateProfileRequest
		setupMock   func(d *userServiceTestDeps, deleteDone chan bool)
		wantErr     bool
		errContains string
	}{
		{
			name: "Success_WithAvatarUpdate",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Name:   "New Name",
				Avatar: createMockFile("test.jpg"),
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				oldAvatar := "old-avatar-url"
				user := &model.User{UserId: userId, AvatarURL: &oldAvatar}

				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.storage.On("UploadFile", ctx, mock.Anything, mock.Anything).Return(nil)
				d.storage.On("GetPublicURL", mock.Anything).Return("https://cdn.com/new.jpg", nil)

				d.storage.On("DeleteFile", mock.Anything, oldAvatar).Return(nil).Run(func(args mock.Arguments) {
					deleteDone <- true
				}).Once()

				d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(nil)
				d.auditRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()
			},
			wantErr: false,
		},
		{
			name: "Error_DBUpdateFailed_ShouldRollbackAndCleanupNewFile",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Avatar: createMockFile("new-avatar.jpg"),
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{UserId: userId}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.storage.On("UploadFile", ctx, mock.Anything, mock.Anything).Return(nil)

				d.storage.On("GetPublicURL", mock.Anything).Return("avatars/generated-id_new-avatar.jpg", nil)

				d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(errors.New("db error"))
				d.mockTx.On("Rollback", ctx).Return(nil)

				d.storage.On("DeleteFile", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					deleteDone <- true
				}).Once()
			},
			wantErr: true,
		},
		{
			name: "Error_UserNotFound",
			req:  dto.UpdateProfileRequest{UserId: "invalid-id"},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				d.userRepo.On("GetByUserId", ctx, "invalid-id").Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "Error_BeginTransactionFailed",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Name:   "New Name",
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{UserId: userId}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.On("Begin", ctx).Return(nil, errors.New("connection pool exhausted"))
			},
			wantErr: true,
		},
		{
			name: "Error_DBUpdateFailed_ShouldRollbackAndCleanupNewFile",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Avatar: createMockFile("new-avatar.jpg"),
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{UserId: userId, AvatarURL: nil}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.storage.On("UploadFile", ctx, mock.Anything, mock.Anything).Return(nil)
				d.storage.On("GetPublicURL", mock.Anything).Return("avatars/new-url.jpg", nil)

				d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(errors.New("deadlock detected"))
				d.mockTx.On("Rollback", ctx).Return(nil)

				d.storage.On("DeleteFile", context.Background(), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					deleteDone <- true
				}).Once()
			},
			wantErr: true,
		},
		{
			name: "Error_AuditLogFailed_ShouldRollback",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Name:   "Ofren Dialsa",
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{UserId: userId}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(nil)

				d.auditRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(errors.New("audit service unavailable"))

				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "Error_CommitFailed_ShouldRollback",
			req: dto.UpdateProfileRequest{
				UserId: userId,
				Name:   "Ofren Dialsa",
			},
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{UserId: userId}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(nil)
				d.auditRepo.On("Create", ctx, d.mockTx, mock.Anything).Return(nil)

				d.mockTx.On("Commit", ctx).Return(errors.New("transaction commit failed"))
				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupUserService(t)
			deleteDone := make(chan bool, 1)

			tt.setupMock(d, deleteDone)

			resp, err := d.svc.UpdateProfile(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.req.Name != "" {
					assert.Equal(t, tt.req.Name, resp.User.Name)
				}
			}

			if tt.name == "Success_WithAvatarUpdate" || tt.name == "Error_DBUpdateFailed_ShouldRollbackAndCleanupNewFile" {
				select {
				case <-deleteDone:
				case <-time.After(200 * time.Millisecond):
					t.Errorf("DeleteFile was expected but not called on %s", tt.name)
				}
			}

			d.userRepo.AssertExpectations(t)
			d.storage.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	userId := "user-123"
	currentPass := "OldPassword123!"
	newPass := "NewSecurePass@2026"

	oldHash, _ := lib.Hash(currentPass)

	tests := []struct {
		name        string
		req         dto.ChangePasswordRequest
		setupMock   func(d *userServiceTestDeps)
		wantErr     bool
		errExpected error
	}{
		{
			name: "Success_WithRevokeOtherSessions",
			req: dto.ChangePasswordRequest{
				UserId:             userId,
				SessionId:          "sess-current",
				Current:            currentPass,
				New:                newPass,
				RevokeOtherSession: lib.Ptr(true),
			},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &oldHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.userRepo.On("UpdatePassword", ctx, nil, userId, mock.AnythingOfType("string")).Return(nil)
				d.sessionRepo.On("RevokeAllExcept", ctx, userId, "sess-current").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error_InvalidCurrentPassword",
			req: dto.ChangePasswordRequest{
				UserId:  userId,
				Current: "wrong-password",
				New:     newPass,
			},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &oldHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrInvalidCurrentPassword,
		},
		{
			name: "Error_WeakPassword",
			req: dto.ChangePasswordRequest{
				UserId:  userId,
				Current: currentPass,
				New:     "123",
			},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &oldHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrWeakPassword,
		},
		{
			name: "Error_SameAsOldPassword",
			req: dto.ChangePasswordRequest{
				UserId:  userId,
				Current: currentPass,
				New:     currentPass,
			},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &oldHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrSamePassword,
		},
		{
			name: "Error_UserHasNoPassword_OAuthUser",
			req:  dto.ChangePasswordRequest{UserId: userId, Current: "any", New: "any"},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: nil}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrUserHasNoPassword,
		},
		{
			name: "Success_RevokeFailed_ShouldStillReturnNil",
			req: dto.ChangePasswordRequest{
				UserId:             userId,
				Current:            currentPass,
				New:                newPass,
				RevokeOtherSession: lib.Ptr(true),
			},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &oldHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.userRepo.On("UpdatePassword", ctx, nil, userId, mock.Anything).Return(nil)
				d.sessionRepo.On("RevokeAllExcept", ctx, userId, mock.Anything).Return(errors.New("redis down"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupUserService(t)
			tt.setupMock(d)

			err := d.svc.ChangePassword(ctx, tt.req)

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
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	ctx := context.Background()
	userId := "user-123"
	password := "Secret123!"

	correctHash, _ := lib.Hash(password)

	tests := []struct {
		name        string
		req         dto.UserDeleteAccountRequest
		setupMock   func(d *userServiceTestDeps)
		wantErr     bool
		errContains string
		errExpected error
	}{
		{
			name: "Success",
			req:  dto.UserDeleteAccountRequest{UserId: userId, Password: password},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &correctHash}

				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.userRepo.On("Delete", ctx, d.mockTx, userId).Return(nil)
				d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, userId).Return(nil)

				d.mockTx.On("Commit", ctx).Return(nil)
				d.mockTx.On("Rollback", ctx).Return(nil).Maybe()
			},
			wantErr: false,
		},
		{
			name: "Error_InvalidPassword",
			req:  dto.UserDeleteAccountRequest{UserId: userId, Password: "wrong-password"},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &correctHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.txStarter.AssertNotCalled(t, "Begin", ctx)
			},
			wantErr:     true,
			errExpected: lib.ErrInvalidPassword,
		},
		{
			name: "Error_BeginTransactionFailed",
			req:  dto.UserDeleteAccountRequest{UserId: userId, Password: password},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &correctHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(nil, errors.New("db connection lost"))
			},
			wantErr:     true,
			errContains: "failed to start transaction",
		},
		{
			name: "Error_DeleteRepoFailed_ShouldRollback",
			req:  dto.UserDeleteAccountRequest{UserId: userId, Password: password},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &correctHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

				d.userRepo.On("Delete", ctx, d.mockTx, userId).Return(errors.New("db error"))
				d.mockTx.On("Rollback", ctx).Return(nil)

				d.sessionRepo.AssertNotCalled(t, "RevokeAllUserSessions", mock.Anything, mock.Anything, mock.Anything)
			},
			wantErr:     true,
			errContains: "failed to delete user account",
		},
		{
			name: "Error_CommitFailed",
			req:  dto.UserDeleteAccountRequest{UserId: userId, Password: password},
			setupMock: func(d *userServiceTestDeps) {
				user := &model.User{UserId: userId, PasswordHash: &correctHash}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
				d.userRepo.On("Delete", ctx, d.mockTx, userId).Return(nil)
				d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, userId).Return(nil)

				d.mockTx.On("Commit", ctx).Return(errors.New("commit error"))
				d.mockTx.On("Rollback", ctx).Return(nil)
			},
			wantErr:     true,
			errContains: "failed to commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupUserService(t)
			tt.setupMock(d)

			err := d.svc.DeleteAccount(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			d.userRepo.AssertExpectations(t)
			d.sessionRepo.AssertExpectations(t)
			d.txStarter.AssertExpectations(t)
			d.mockTx.AssertExpectations(t)
		})
	}
}

func TestDeleteAvatar(t *testing.T) {
	ctx := context.Background()
	userId := "user-123"

	tests := []struct {
		name        string
		userId      string
		setupMock   func(d *userServiceTestDeps, deleteDone chan bool)
		wantErr     bool
		errExpected error
	}{
		{
			name:   "Success_WithExistingAvatar",
			userId: userId,
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				oldAvatar := "avatars/old-photo.jpg"
				user := &model.User{
					UserId:    userId,
					AvatarURL: &oldAvatar,
				}

				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.userRepo.On("UpdateAvatar", ctx, nil, userId, (*string)(nil)).Return(nil)

				d.storage.On("DeleteFile", mock.Anything, oldAvatar).Return(nil).Run(func(args mock.Arguments) {
					deleteDone <- true
				}).Once()
			},
			wantErr: false,
		},
		{
			name:   "Success_UserAlreadyHasNoAvatar",
			userId: userId,
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				user := &model.User{
					UserId:    userId,
					AvatarURL: nil,
				}
				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

				d.userRepo.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything)
				d.storage.AssertNotCalled(t, "DeleteFile", mock.Anything, mock.Anything)
			},
			wantErr: false,
		},
		{
			name:   "Error_UserNotFound",
			userId: "ghost-user",
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				d.userRepo.On("GetByUserId", ctx, "ghost-user").Return(nil, nil)
			},
			wantErr:     true,
			errExpected: lib.ErrUserNotFound,
		},
		{
			name:   "Error_DatabaseUpdateFailed_ShouldNotDeleteFile",
			userId: userId,
			setupMock: func(d *userServiceTestDeps, deleteDone chan bool) {
				oldPath := "path/to/avatar.png"
				user := &model.User{UserId: userId, AvatarURL: &oldPath}

				d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
				d.userRepo.On("UpdateAvatar", ctx, nil, userId, (*string)(nil)).Return(errors.New("db connection error"))

				d.storage.AssertNotCalled(t, "DeleteFile", mock.Anything, mock.Anything)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := setupUserService(t)
			deleteDone := make(chan bool, 1)

			tt.setupMock(d, deleteDone)

			err := d.svc.DeleteAvatar(ctx, tt.userId)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.name == "Success_WithExistingAvatar" {
				select {
				case <-deleteDone:
				case <-time.After(200 * time.Millisecond):
					t.Errorf("DeleteFile was expected but not called for %s", tt.name)
				}
			}

			d.userRepo.AssertExpectations(t)
			d.storage.AssertExpectations(t)
		})
	}
}
