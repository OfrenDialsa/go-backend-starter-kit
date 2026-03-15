package dto

import (
	"mime/multipart"
	"time"
)

type GetMeResponse struct {
	User UserProfileData `json:"user"`
}

type UpdateProfileRequest struct {
	UserId    string
	Ip        string
	Ua        string
	Name      string                `json:"name" form:"name"`
	Avatar    *multipart.FileHeader `form:"avatar"`
	AvatarURL string                `form:"-"`
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
	EmailVerifiedAt time.Time  `json:"email_verified_at"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ChangePasswordRequest struct {
	UserId             string `form:"-"`
	SessionId          string `form:"-"`
	Current            string `json:"current" binding:"required"`
	New                string `json:"new" binding:"required"`
	RevokeOtherSession *bool  `json:"revoke_other_session" binding:"required"`
}

type UserDeleteAccountRequest struct {
	UserId   string `form:"-"`
	Password string `json:"password"`
}
