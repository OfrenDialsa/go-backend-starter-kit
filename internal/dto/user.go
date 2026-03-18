package dto

import (
	"mime/multipart"
	"time"
)

type GetMeResponse struct {
	User UserProfileData `json:"user"`
}

type UpdateProfileRequest struct {
	UserId    string                `json:"-" swaggerignore:"true"`
	Ip        string                `json:"-" swaggerignore:"true"`
	Ua        string                `json:"-" swaggerignore:"true"`
	Username  string                `json:"username" form:"username"`
	Name      string                `json:"name" form:"name"`
	Avatar    *multipart.FileHeader `form:"avatar"`
	AvatarURL string                `form:"-" swaggerignore:"true"`
}

type UpdateProfileResponse struct {
	User UpdateProfileData `json:"user"`
}

type UpdateProfileData struct {
	UserId    string  `json:"user_id"`
	Email     string  `json:"email"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

type UserProfileData struct {
	UserId          string     `json:"user_id"`
	Email           string     `json:"email"`
	Username        string     `json:"username"`
	Name            string     `json:"name"`
	AvatarURL       *string    `json:"avatar_url"`
	Status          string     `json:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ChangePasswordRequest struct {
	UserId             string `json:"-" swaggerignore:"true"`
	SessionId          string `json:"-" swaggerignore:"true"`
	Current            string `json:"current" binding:"required"`
	New                string `json:"new" binding:"required"`
	RevokeOtherSession *bool  `json:"revoke_other_session" binding:"required"`
}

type UserDeleteAccountRequest struct {
	UserId   string `json:"-" swaggerignore:"true"`
	Password string `json:"password"`
}
