package service

import (
	"bytes"
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/external/storage"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/model"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type userServiceImpl struct {
	env         *config.EnvironmentVariable
	txStarter   TxStarter
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	auditRepo   repository.AuditLogRepository
	storage     storage.StorageService
}

func NewUserService(env *config.EnvironmentVariable, txStarter TxStarter, userRepo repository.UserRepository, sessionRepo repository.SessionRepository,
	auditRepo repository.AuditLogRepository, storage storage.StorageService) UserService {
	return &userServiceImpl{
		env:         env,
		txStarter:   txStarter,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		storage:     storage,
	}
}

func (u *userServiceImpl) GetMe(ctx context.Context, userId string) (*dto.GetMeResponse, error) {
	user, err := u.userRepo.GetByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, lib.ErrUserNotFound
	}

	return &dto.GetMeResponse{
		User: dto.UserProfileData{
			UserId:          user.UserId,
			Email:           user.Email,
			Username:        user.Username,
			Name:            user.Name,
			AvatarURL:       user.AvatarURL,
			Status:          user.Status,
			EmailVerifiedAt: user.EmailVerifiedAt,
			LastLoginAt:     user.LastLoginAt,
			CreatedAt:       user.CreatedAt,
		},
	}, nil
}

func (u *userServiceImpl) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (*dto.UpdateProfileResponse, error) {
	user, err := u.userRepo.GetByUserId(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, lib.ErrUserNotFound
	}

	if req.Username != "" {
		existingUser, err := u.userRepo.GetByUsername(ctx, req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if existingUser != nil && existingUser.UserId != req.UserId {
			return nil, lib.ErrUsernameNotAvailable
		}

		user.Username = req.Username
	}

	var oldPath string
	var uploadedFile string
	var isSuccess bool

	if user.AvatarURL != nil {
		oldPath = *user.AvatarURL
	}

	tx, err := u.txStarter.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if !isSuccess {
			if err := tx.Rollback(ctx); err != nil {
				log.Error().Err(err).Msg("transaction rollback failed")
			}

			if uploadedFile != "" {
				go func(path string) {
					if err := u.storage.DeleteFile(context.Background(), path); err != nil {
						log.Error().
							Err(err).
							Str("file_path", path).
							Msg("failed to delete uploaded file")
					}
				}(uploadedFile)
			}
		}
	}()

	if req.Avatar != nil {
		file, err := req.Avatar.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open avatar file: %w", err)
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			return nil, fmt.Errorf("error copying file to buffer: %w", err)
		}

		fileName := fmt.Sprintf("avatars/%s_%s", uuid.NewString(), req.Avatar.Filename)

		if err := u.storage.UploadFile(ctx, fileName, buf); err != nil {
			return nil, fmt.Errorf("failed to upload avatar: %w", err)
		}

		uploadedFile = fileName

		publicURL, err := u.storage.GetPublicURL(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate public url: %w", err)
		}

		user.AvatarURL = &publicURL
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	user.UpdatedAt = time.Now()

	if err := u.userRepo.Update(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	auditLog := &model.AuditLog{
		UserID:       &req.UserId,
		Action:       "UPDATE_PROFILE",
		ResourceType: "users",
		ResourceID:   req.UserId,
		Details:      []byte(`{"message":"update profile"}`),
		CreatedAt:    time.Now(),
	}

	if err := u.auditRepo.Create(ctx, tx, auditLog); err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	isSuccess = true

	if uploadedFile != "" && oldPath != "" {
		go func(path string) {
			if err := u.storage.DeleteFile(context.Background(), path); err != nil {
				log.Error().
					Err(err).
					Str("file_path", path).
					Msg("failed to delete old avatar")
			}
		}(oldPath)
	}

	var avatarURL *string
	if user.AvatarURL != nil {
		avatarURL = user.AvatarURL
	}

	return &dto.UpdateProfileResponse{
		User: dto.UpdateProfileData{
			UserId:    user.UserId,
			Email:     user.Email,
			Username:  user.Username,
			Name:      user.Name,
			AvatarURL: avatarURL,
		},
	}, nil
}

func (u *userServiceImpl) ChangePassword(ctx context.Context, req dto.ChangePasswordRequest) error {
	user, err := u.userRepo.GetByUserId(ctx, req.UserId)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return lib.ErrUserNotFound
	}

	if user.PasswordHash == nil {
		return lib.ErrUserHasNoPassword
	}

	if err := lib.Verify(*user.PasswordHash, req.Current); err != nil {
		return lib.ErrInvalidCurrentPassword
	}

	if !lib.IsValidPassword(req.New) {
		return lib.ErrWeakPassword
	}

	if err := lib.Verify(*user.PasswordHash, req.New); err == nil {
		return lib.ErrSamePassword
	}

	hashedPassword, err := lib.Hash(req.New)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err = u.userRepo.UpdatePassword(ctx, nil, req.UserId, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if req.RevokeOtherSession != nil {
		if *req.RevokeOtherSession {
			err = u.sessionRepo.RevokeAllExcept(ctx, req.UserId, req.SessionId)
			if err != nil {
				log.Warn().
					Err(err).
					Str("user_id", req.UserId).
					Msg("failed to revoke other sessions")
			}
		}
	}

	return nil
}

func (u *userServiceImpl) DeleteAccount(ctx context.Context, req dto.UserDeleteAccountRequest) error {
	user, err := u.userRepo.GetByUserId(ctx, req.UserId)
	if err != nil {
		return lib.ErrUserNotFound
	}

	if err := lib.Verify(*user.PasswordHash, req.Password); err != nil {
		return lib.ErrInvalidPassword
	}

	tx, err := u.txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = u.userRepo.Delete(ctx, tx, req.UserId)
	if err != nil {
		return fmt.Errorf("failed to delete user account: %w", err)
	}

	err = u.sessionRepo.RevokeAllUserSessions(ctx, tx, req.UserId)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (u *userServiceImpl) DeleteAvatar(ctx context.Context, userId string) error {
	user, err := u.userRepo.GetByUserId(ctx, userId)
	if err != nil || user == nil {
		return lib.ErrUserNotFound
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return nil
	}

	oldPath := *user.AvatarURL

	err = u.userRepo.UpdateAvatar(ctx, nil, userId, nil)
	if err != nil {
		return err
	}

	go func(path string) {
		u.storage.DeleteFile(context.Background(), path)
	}(oldPath)

	return nil
}
