package handler

import (
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type UserHandlerImpl struct {
	userService service.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService service.UserService) UserHandler {
	return &UserHandlerImpl{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetMe godoc
// @Summary      Get current user profile
// @Description  Fetches the detailed profile of the authenticated user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  lib.APIResponse{data=dto.GetMeResponse} "Successfully retrieved profile"
// @Failure      401  {object}  lib.HTTPError "Unauthorized: Missing or invalid token"
// @Failure      403  {object}  lib.HTTPError "Forbidden: User is not associated with the requested organization"
// @Failure      404  {object}  lib.HTTPError "Not Found: User record does not exist"
// @Failure      500  {object}  lib.HTTPError "Internal Server Error: Something went wrong on our end"
// @Router       /api/v1/users/me [get]
func (u *UserHandlerImpl) GetMe(ctx *gin.Context) {
	claims := ctx.MustGet("user").(*lib.JWTClaims)
	if claims == nil {
		lib.RespondError(ctx, http.StatusUnauthorized, "unauthorized: missing context", nil)
		return
	}

	data, err := u.userService.GetMe(ctx, claims.UserId)
	if err != nil {
		log.Error().Err(err).Str("userId", claims.UserId).Msg("Error in GetMe")

		if err == lib.ErrorMessageUserNotFound {
			lib.RespondError(ctx, http.StatusNotFound, lib.ErrUserNotFound, nil)
			return
		}

		lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Get profile success", data)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates the name and avatar URL of the authenticated user.
// @Tags         Users
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        name    formData  string  false  "New name of the user"
// @Param        avatar  formData  file    false  "User avatar image file"
// @Success      200  {object}  lib.APIResponse{data=dto.UpdateProfileResponse} "Profile updated successfully"
// @Failure      400  {object}  lib.HTTPError "Bad Request: Invalid input"
// @Failure      401  {object}  lib.HTTPError "Unauthorized"
// @Failure      404  {object}  lib.HTTPError "Not Found"
// @Failure      500  {object}  lib.HTTPError "Internal Server Error"
// @Router       /api/v1/users/me [put]
func (u *UserHandlerImpl) UpdateProfile(ctx *gin.Context) {
	claims := ctx.MustGet("user").(*lib.JWTClaims)
	if claims == nil {
		lib.RespondError(ctx, http.StatusUnauthorized, "unauthorized: missing context", nil)
		return
	}

	var req dto.UpdateProfileRequest
	if err := ctx.ShouldBind(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	req.UserId = claims.UserId
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" && req.Avatar == nil {
		lib.RespondError(ctx, http.StatusBadRequest, "At least name or avatar must be provided", nil)
		return
	}

	if req.Name != "" {
		req.Name = strings.Join(strings.Fields(req.Name), " ")
	}

	if req.Avatar != nil {
		file, err := req.Avatar.Open()
		if err != nil {
			lib.RespondError(ctx, http.StatusInternalServerError, "Failed to open avatar file", err)
			return
		}
		defer file.Close()

		buf := make([]byte, 512)
		_, err = file.Read(buf)
		if err != nil {
			lib.RespondError(ctx, http.StatusInternalServerError, "Failed to read avatar file", err)
			return
		}

		if _, err := file.Seek(0, 0); err != nil {
			lib.RespondError(ctx, http.StatusInternalServerError, "Failed to reset file pointer", err)
			return
		}

		contentType := http.DetectContentType(buf)

		allowedTypes := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/webp": true,
		}

		if !allowedTypes[contentType] {
			lib.RespondError(ctx, http.StatusBadRequest, "This file type is not allowed. please upload jpeg, png, or webp.", nil)
			return
		}

		if req.Avatar.Size > 2*1024*1024 {
			lib.RespondError(ctx, http.StatusBadRequest, "Avatar image is too large (max 2MB)", nil)
			return
		}
	}

	req.Ip = ctx.ClientIP()
	req.Ua = ctx.Request.UserAgent()

	data, err := u.userService.UpdateProfile(ctx.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Str("userId", claims.UserId).Msg("Error in UpdateProfile Service")

		if err == lib.ErrorMessageUserNotFound {
			lib.RespondError(ctx, http.StatusNotFound, err.Error(), err)
			return
		}

		lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Update profile success", data)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Allows an authenticated user to change their password and optionally revoke other active sessions.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.ChangePasswordRequest  true  "Change Password Request"
// @Success      200   {object}  lib.APIResponse            "Password changed successfully"
// @Failure      400   {object}  lib.HTTPError              "Bad Request: Validation error or password mismatch"
// @Failure      401   {object}  lib.HTTPError              "Unauthorized: Missing or invalid token"
// @Failure      500   {object}  lib.HTTPError              "Internal Server Error"
// @Router       /api/v1/users/me/password [put]
func (u *UserHandlerImpl) ChangePassword(ctx *gin.Context) {
	claims := ctx.MustGet("user").(*lib.JWTClaims)
	if claims == nil {
		lib.RespondError(ctx, http.StatusUnauthorized, "unauthorized: missing context", nil)
		return
	}

	var req dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	req.UserId = claims.UserId
	req.SessionId = claims.SessionId
	err := u.userService.ChangePassword(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("userId", req.UserId).Msg("Error in ChangePassword")

		switch err {
		case lib.ErrorMessageUserHasNoPassword:
			lib.RespondErrorWithCode(ctx, http.StatusBadRequest, lib.ErrUserHasNoPassword, lib.CodeUserHasNoPassword)
		case lib.ErrorMessageInvalidCurrentPassword:
			lib.RespondErrorWithCode(ctx, http.StatusBadRequest, lib.ErrInvalidCurrentPassword, lib.CodeInvalidCurrentPassword)
		case lib.ErrorMessageWeakPassword:
			lib.RespondErrorWithCode(ctx, http.StatusBadRequest, lib.ErrWeakPassword, lib.CodeWeakPassword)
		case lib.ErrorMessageSamePassword:
			lib.RespondErrorWithCode(ctx, http.StatusBadRequest, lib.ErrSamePassword, lib.CodeSamePassword)
		case lib.ErrorMessageUserNotFound:
			lib.RespondError(ctx, http.StatusNotFound, lib.ErrUserNotFound, nil)
		default:
			lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", nil)
		}
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Password updated successfully", nil)
}

// DeleteAccount godoc
// @Summary      Delete user account
// @Description  Deletes the authenticated user's account.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UserDeleteAccountRequest true "Delete account request"
// @Success      200  {object}  lib.APIResponse "Account deleted successfully"
// @Failure      400  {object}  lib.HTTPError "Invalid password or ownership transfer required"
// @Failure      404  {object}  lib.HTTPError "User not found"
// @Failure      500  {object}  lib.HTTPError "Internal server error"
// @Router       /api/v1/users/me [delete]
func (u *UserHandlerImpl) DeleteAccount(ctx *gin.Context) {
	claims := ctx.MustGet("user").(*lib.JWTClaims)
	if claims == nil {
		lib.RespondError(ctx, http.StatusUnauthorized, "unauthorized: missing context", nil)
		return
	}

	var req dto.UserDeleteAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	req.UserId = claims.UserId
	err := u.userService.DeleteAccount(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("userId", claims.UserId).Msg("Error in DeleteAccount")

		switch err {
		case lib.ErrorMessageInvalidPassword:
			lib.RespondErrorWithCode(ctx, http.StatusBadRequest, lib.ErrInvalidPassword, lib.CodeInvalidPassword)
		case lib.ErrorMessageUserNotFound:
			lib.RespondError(ctx, http.StatusNotFound, lib.ErrUserNotFound, nil)
		default:
			lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", nil)
		}
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Account deleted successfully", nil)
}
