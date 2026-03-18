package repository

import (
	"context"
	"fmt"
	"github/OfrenDialsa/go-gin-starter/database"
	"time"

	"github/OfrenDialsa/go-gin-starter/internal/model"

	"github.com/jackc/pgx/v5"
)

type userRepositoryImpl struct {
	db *database.WrapDB
}

func NewUserRepository(db *database.WrapDB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) GetByEmailOrUsername(ctx context.Context, email, username string) (*model.User, error) {
	query := `
		SELECT id, user_id, email, username, password_hash, name, avatar_url, status,
		       role, email_verified_at, last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE (email = $1 OR username = $2) AND deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := r.db.Database.Conn.QueryRow(ctx, query, email, username).Scan(
		&user.Id,
		&user.UserId,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.Status,
		&user.Role,
		&user.EmailVerifiedAt,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, user_id, email, username, password_hash, name, avatar_url, status,
		       role, email_verified_at, last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := r.db.Database.Conn.QueryRow(ctx, query, email).Scan(
		&user.Id,
		&user.UserId,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.Status,
		&user.Role,
		&user.EmailVerifiedAt,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, user_id, email, username, password_hash, name, avatar_url, status,
		       role, email_verified_at, last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := r.db.Database.Conn.QueryRow(ctx, query, username).Scan(
		&user.Id,
		&user.UserId,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.Status,
		&user.Role,
		&user.EmailVerifiedAt,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) GetByUserId(ctx context.Context, userId string) (*model.User, error) {
	query := `
		SELECT id, user_id, email, username, password_hash, name, avatar_url, status,
		       email_verified_at, last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	var user model.User
	err := r.db.Database.Conn.QueryRow(ctx, query, userId).Scan(
		&user.Id,
		&user.UserId,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.Status,
		&user.EmailVerifiedAt,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by Id: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, user *model.User) error {
	query := `
		INSERT INTO users (user_id, email, username, password_hash, name, avatar_url, role, status, email_verified_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`

	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query,
			user.UserId,
			user.Email,
			user.Username,
			user.PasswordHash,
			user.Name,
			user.AvatarURL,
			user.Role,
			user.Status,
			user.EmailVerifiedAt,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.Id)
	} else {
		err = r.db.Database.Conn.QueryRow(ctx, query,
			user.UserId,
			user.Email,
			user.Username,
			user.PasswordHash,
			user.Name,
			user.AvatarURL,
			user.Status,
			user.EmailVerifiedAt,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.Id)
	}

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) Update(ctx context.Context, tx pgx.Tx, user *model.User) error {
	query := `
		UPDATE users
		SET email = $1, username = $2, password_hash = $3, name = $4, avatar_url = $5,
		    status = $6, last_login_at = $7, updated_at = $8
		WHERE user_id = $9 AND deleted_at IS NULL
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			user.Email,
			user.Username,
			user.PasswordHash,
			user.Name,
			user.AvatarURL,
			user.Status,
			user.LastLoginAt,
			user.UpdatedAt,
			user.UserId,
		)

	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query,
			user.Email,
			user.Username,
			user.PasswordHash,
			user.Name,
			user.AvatarURL,
			user.Status,
			user.LastLoginAt,
			user.UpdatedAt,
			user.UserId,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) UpdateAvatar(ctx context.Context, tx pgx.Tx, userId string, avatarUrl *string) error {
	query := `
		UPDATE users 
		SET avatar_url = $1, updated_at = $2 
		WHERE user_id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	if tx != nil {
		_, err := tx.Exec(ctx, query, avatarUrl, now, userId)
		return err
	}

	_, err := r.db.Database.Conn.Exec(ctx, query, avatarUrl, now, userId)
	return err
}

func (r *userRepositoryImpl) UpdateVerifiedEmail(ctx context.Context, tx pgx.Tx, userId string) error {
	query := `
        UPDATE users 
        SET email_verified_at = $1, 
            updated_at = $2 
        WHERE user_id = $3 AND deleted_at IS NULL
    `

	now := time.Now()

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, now, now, userId)
	} else {
		_, err = r.db.Database.Conn.Exec(ctx, query, now, now, userId)
	}

	return err
}

func (r *userRepositoryImpl) UpdateLastLogin(ctx context.Context, userId string, time time.Time) error {
	query := `
		UPDATE users
		SET last_login_at = $1, updated_at = $2
		WHERE user_id = $3
	`
	_, err := r.db.Database.Conn.Exec(ctx, query, time, time, userId)
	return err
}

func (r *userRepositoryImpl) UpdatePassword(ctx context.Context, userId string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1
		WHERE user_id = $2 AND deleted_at IS NULL
	`

	_, err := r.db.Database.Conn.Exec(ctx, query, passwordHash, userId)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) Delete(ctx context.Context, tx pgx.Tx, userId string) error {
	query := `
        UPDATE users 
        SET deleted_at = NOW(), updated_at = NOW() 
        WHERE user_id = $1 AND deleted_at IS NULL
    `
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, userId)
	} else {

		_, err = r.db.Database.Conn.Exec(ctx, query, userId)
	}

	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	return nil
}

func (r *userRepositoryImpl) CheckEmailExist(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE email = $1 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.Database.Conn.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

func (r *userRepositoryImpl) CheckUsernameExist(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE username = $1 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.Database.Conn.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}
