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

type userTestDeps struct {
	userRepo    *mocks.UserRepository
	sessionRepo *mocks.SessionRepository
	auditRepo   *mocks.AuditLogRepository
	txStarter   *mocks.TxStarter
	mockTx      *mocks.Tx
	storage     *mocks.StorageService
	svc         service.UserService
}

func setupUserService(t *testing.T) *userTestDeps {
	t.Helper()
	d := &userTestDeps{
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

// ===================== GetMe Tests =====================

func TestGetMe_Success(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-123"

	user := &model.User{UserId: userId, Email: "ofren@example.com", Name: "Ofren"}
	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

	resp, err := d.svc.GetMe(ctx, userId)

	assert.NoError(t, err)
	assert.Equal(t, userId, resp.User.UserId)
	assert.Equal(t, "Ofren", resp.User.Name)
}

func TestGetMe_UserNotFound(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "non-existent-id"
	d.userRepo.On("GetByUserId", ctx, userId).Return(nil, nil)

	resp, err := d.svc.GetMe(ctx, userId)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, lib.ErrUserNotFound)
}

func TestGetMe_DatabaseError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-123"

	d.userRepo.On("GetByUserId", ctx, userId).Return(nil, errors.New("database connection failed"))

	resp, err := d.svc.GetMe(ctx, userId)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database connection failed")
}

// ===================== UpdateProfile Tests =====================
func TestUpdateProfile_Success_WithAvatar(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-123"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "test.jpg")
	_, _ = part.Write([]byte("fake-image-content"))
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(1024)
	mockFileHeader := form.File["avatar"][0]

	deleteDone := make(chan bool, 1)

	oldAvatar := "old-avatar-url"
	user := &model.User{
		UserId:    userId,
		AvatarURL: &oldAvatar,
		Email:     "ofren@example.com",
	}

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

	req := dto.UpdateProfileRequest{
		UserId: userId,
		Name:   "Updated Name",
		Avatar: mockFileHeader,
	}

	resp, err := d.svc.UpdateProfile(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Updated Name", resp.User.Name)

	select {
	case <-deleteDone:
	case <-time.After(100 * time.Millisecond):
	}
}

func TestUpdateProfile_TxFail_RollbackFile(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "u1"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "test.jpg")
	_, _ = part.Write([]byte("fake-content"))
	writer.Close()
	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(1024)
	mockFileHeader := form.File["avatar"][0]

	user := &model.User{UserId: userId}
	deleteDone := make(chan bool, 1)
	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
	d.storage.On("UploadFile", ctx, mock.Anything, mock.Anything).Return(nil)
	d.storage.On("GetPublicURL", mock.Anything).Return("new-url", nil)

	d.userRepo.On("Update", ctx, d.mockTx, mock.Anything).Return(errors.New("db error"))
	d.mockTx.On("Rollback", ctx).Return(nil)

	d.storage.On("DeleteFile", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		deleteDone <- true
	}).Once()

	req := dto.UpdateProfileRequest{
		UserId: userId,
		Avatar: mockFileHeader,
	}

	_, err := d.svc.UpdateProfile(ctx, req)

	assert.Error(t, err)

	select {
	case <-deleteDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DeleteFile was not called within timeout")
	}
}

// ===================== ChangePassword Tests =====================
func TestChangePassword_Success(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "u1"

	oldHash := hashPassword(t)
	user := &model.User{UserId: userId, PasswordHash: &oldHash}

	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)
	d.userRepo.On("UpdatePassword", ctx, userId, mock.Anything).Return(nil)
	d.sessionRepo.On("RevokeAllExcept", ctx, userId, "sess-current").Return(nil)

	err := d.svc.ChangePassword(ctx, dto.ChangePasswordRequest{
		UserId:             userId,
		SessionId:          "sess-current",
		Current:            "password123",
		New:                "NewPassword@123",
		RevokeOtherSession: lib.Ptr(true),
	})

	assert.NoError(t, err)
}

func TestChangePassword_WeakPassword(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}
	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)

	err := d.svc.ChangePassword(ctx, dto.ChangePasswordRequest{
		UserId:  "u1",
		Current: "password123",
		New:     "123",
	})

	assert.Error(t, err)
	assert.Equal(t, lib.ErrWeakPassword, err)
}

// ===================== DeleteAccount Tests =====================

func TestDeleteAccount_Success(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
	d.userRepo.On("Delete", ctx, d.mockTx, "u1").Return(nil)
	d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, "u1").Return(nil)
	d.mockTx.On("Commit", ctx).Return(nil)
	d.mockTx.On("Rollback", ctx).Return(nil).Maybe()

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "password123",
	})

	assert.NoError(t, err)
}

func TestDeleteAccount_InvalidPassword(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "wrongpassword",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, lib.ErrInvalidPassword)

	d.txStarter.AssertNotCalled(t, "Begin", ctx)
}

func TestDeleteAccount_BeginTxError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)

	d.txStarter.On("Begin", ctx).Return(nil, errors.New("db connection lost"))

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start transaction")
}

func TestDeleteAccount_DeleteRepoError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)

	d.userRepo.On("Delete", ctx, d.mockTx, "u1").Return(errors.New("constraint violation"))
	d.mockTx.On("Rollback", ctx).Return(nil)

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user account")

	d.sessionRepo.AssertNotCalled(t, "RevokeAllUserSessions", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeleteAccount_RevokeSessionError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
	d.userRepo.On("Delete", ctx, d.mockTx, "u1").Return(nil)

	d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, "u1").Return(errors.New("internal error"))
	d.mockTx.On("Rollback", ctx).Return(nil)

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to revoke user sessions")
}

func TestDeleteAccount_CommitError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()

	hash := hashPassword(t)
	user := &model.User{UserId: "u1", PasswordHash: &hash}

	d.userRepo.On("GetByUserId", ctx, "u1").Return(user, nil)
	d.txStarter.On("Begin", ctx).Return(d.mockTx, nil)
	d.userRepo.On("Delete", ctx, d.mockTx, "u1").Return(nil)
	d.sessionRepo.On("RevokeAllUserSessions", ctx, d.mockTx, "u1").Return(nil)

	d.mockTx.On("Commit", ctx).Return(errors.New("commit failed"))
	d.mockTx.On("Rollback", ctx).Return(nil)

	err := d.svc.DeleteAccount(ctx, dto.UserDeleteAccountRequest{
		UserId:   "u1",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
}

// ===================== DeleteAvatar Tests =====================
func TestDeleteAvatar_Success(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-123"
	oldAvatar := "avatars/old-photo.jpg"

	deleteDone := make(chan bool, 1)

	user := &model.User{
		UserId:    userId,
		AvatarURL: &oldAvatar,
	}

	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

	d.userRepo.On("UpdateAvatar", ctx, nil, userId, (*string)(nil)).Return(nil)

	d.storage.On("DeleteFile", mock.Anything, oldAvatar).Return(nil).Run(func(args mock.Arguments) {
		deleteDone <- true
	}).Once()

	err := d.svc.DeleteAvatar(ctx, userId)

	assert.NoError(t, err)

	select {
	case <-deleteDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DeleteFile was not called within timeout")
	}
}

func TestDeleteAvatar_UserNotFound(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "invalid-user"

	d.userRepo.On("GetByUserId", ctx, userId).Return(nil, nil)

	err := d.svc.DeleteAvatar(ctx, userId)

	assert.ErrorIs(t, err, lib.ErrUserNotFound)
}

func TestDeleteAvatar_NoAvatar(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-no-avatar"

	user := &model.User{
		UserId:    userId,
		AvatarURL: nil,
	}

	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

	err := d.svc.DeleteAvatar(ctx, userId)

	assert.NoError(t, err)
	d.userRepo.AssertNotCalled(t, "UpdateAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDeleteAvatar_UpdateError(t *testing.T) {
	d := setupUserService(t)
	ctx := context.Background()
	userId := "user-123"
	oldAvatar := "some-path"

	user := &model.User{
		UserId:    userId,
		AvatarURL: &oldAvatar,
	}

	d.userRepo.On("GetByUserId", ctx, userId).Return(user, nil)

	d.userRepo.On("UpdateAvatar", ctx, nil, userId, (*string)(nil)).Return(errors.New("db connection error"))

	err := d.svc.DeleteAvatar(ctx, userId)

	assert.Error(t, err)
	d.storage.AssertNotCalled(t, "DeleteFile", mock.Anything, mock.Anything)
}
