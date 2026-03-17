package handler

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/dto"
	"github/OfrenDialsa/go-gin-starter/internal/service"
	"github/OfrenDialsa/go-gin-starter/lib"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandlerImpl struct {
	env         *config.EnvironmentVariable
	authService service.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(env *config.EnvironmentVariable, authService service.AuthService) AuthHandler {
	return &AuthHandlerImpl{
		env:         env,
		authService: authService,
		validator:   validator.New(),
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register Request"
// @Success 201 {object} lib.APIResponse{data=dto.RegisterResponse}
// @Failure 400 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router /api/v1/auth/register [post]
func (h *AuthHandlerImpl) Register(ctx *gin.Context) {
	var req dto.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, "Validation failed", parseValidationErrors(err))
		return
	}

	resp, err := h.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case lib.ErrorMessageEmailExists:
			lib.RespondError(ctx, http.StatusBadRequest, lib.ErrEmailAlreadyExists, nil)
			return
		case lib.ErrorMessageUsernameNotAvailable:
			lib.RespondError(ctx, http.StatusBadRequest, lib.ErrUsernameNotAvailable, nil)
		default:
			lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", nil)
		}
		return
	}

	lib.RespondSuccess(ctx, http.StatusCreated, lib.MsgRegistrationSuccess, resp)
}

// Login handles user login
// @Summary Login user
// @Description Login user using email or username
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login Request"
// @Success 200 {object} lib.APIResponse{data=dto.LoginResponse}
// @Failure 400 {object} lib.HTTPError
// @Failure 401 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router /api/v1/auth/login [post]
func (h *AuthHandlerImpl) Login(ctx *gin.Context) {
	var req dto.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, "Validation failed", parseValidationErrors(err))
		return
	}

	req.IPAddress = ctx.ClientIP()
	req.UserAgent = ctx.Request.UserAgent()

	resp, err := h.authService.Login(ctx.Request.Context(), req)
	if err != nil {
		if err == lib.ErrorMessageInvalidCredentials {
			lib.RespondError(ctx, http.StatusUnauthorized, "Invalid email or password", err)
			return
		}

		lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, lib.MsgLoginSuccess, resp)
}

// RefreshToken handles token refresh request
// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token Request"
// @Success 200 {object} lib.APIResponse{data=dto.RefreshTokenResponse}
// @Failure 401 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandlerImpl) RefreshToken(ctx *gin.Context) {
	var req dto.RefreshTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	resp, err := h.authService.RefreshToken(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err.Error() {
		case "invalid refresh token", "token already used: security breach detected":
			lib.RespondError(ctx, http.StatusUnauthorized, err.Error(), err)
		case "session not found":
			lib.RespondError(ctx, http.StatusNotFound, "session not found", err)
		case "session revoked or expired":
			lib.RespondError(ctx, http.StatusUnauthorized, "session is no longer active, please login again", err)
		default:
			lib.RespondError(ctx, http.StatusInternalServerError, "internal server error", err)
		}
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Token refreshed successfully", resp)
}

// Logout handles user logout
// @Summary Logout user
// @Description Revoke current user session and invalidate the refresh token in database
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} lib.APIResponse
// @Failure 401 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router /api/v1/auth/logout [post]
func (h *AuthHandlerImpl) Logout(ctx *gin.Context) {
	claims := ctx.MustGet("user").(*lib.JWTClaims)
	if claims == nil {
		lib.RespondError(ctx, http.StatusUnauthorized, "unauthorized: missing context", nil)
		return
	}

	err := h.authService.Logout(ctx.Request.Context(), claims.SessionId)
	if err != nil {
		if err.Error() == "session invalid or revoked" {
			lib.RespondError(ctx, http.StatusUnauthorized, "Session already revoked", err)
			return
		}

		lib.RespondError(ctx, http.StatusInternalServerError, "Failed to process logout", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Logged out successfully", nil)
}

// ResetPassword handles password reset using token
// @Summary Reset user password
// @Description Reset user password using a reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Reset Password Request"
// @Success 200 {object} lib.APIResponse
// @Failure 400 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router  /api/v1/auth/reset-password [post]
func (h *AuthHandlerImpl) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	err := h.authService.ResetPassword(ctx.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		if err.Error() == lib.CodeInvalidResetToken {
			lib.RespondError(ctx, http.StatusBadRequest, lib.ErrInvalidResetToken, err)
			return
		}

		lib.RespondError(ctx, http.StatusInternalServerError, "Failed to reset password", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, lib.MsgPasswordResetSuccess, nil)
}

// ForgotPassword handles forgot password request
// @Summary Forgot password
// @Description Send reset password link to email
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 {object} lib.APIResponse{message=string}
// @Failure 400 {object} lib.HTTPError
// @Failure 429 {object} lib.HTTPError
// @Failure 500 {object} lib.HTTPError
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandlerImpl) ForgotPassword(ctx *gin.Context) {
	var req dto.ForgotPasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		lib.RespondValidationError(ctx, http.StatusBadRequest, lib.ErrBadPayload, parseValidationErrors(err))
		return
	}

	ipAddress := ctx.ClientIP()
	userAgent := ctx.Request.UserAgent()

	err := h.authService.ForgotPassword(ctx.Request.Context(), req.Email, userAgent, ipAddress)
	if err != nil {
		lib.RespondError(ctx, http.StatusInternalServerError, "Failed to process forgot password", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, lib.MsgPasswordForgotSuccess, nil)
}

// CheckEmail handles email availability check
// @Summary Check email availability
// @Description Check if an email is already registered
// @Tags Auth
// @Param email query string true "Email to check"
// @Success 200 {object} lib.APIResponse{data=map[string]bool}
// @Failure 400 {object} lib.HTTPError
// @Router /api/v1/auth/check-email [get]
func (h *AuthHandlerImpl) CheckEmail(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		lib.RespondError(ctx, http.StatusBadRequest, "Email query parameter is required", nil)
		return
	}

	exists, err := h.authService.CheckEmail(ctx.Request.Context(), email)
	if err != nil {
		lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Email availability checked", gin.H{
		"exists": exists,
	})
}

// CheckUsername handles username availability check
// @Summary Check username availability
// @Description Check if a username is already taken
// @Tags Auth
// @Param username query string true "Username to check"
// @Success 200 {object} lib.APIResponse{data=map[string]bool}
// @Failure 400 {object} lib.HTTPError
// @Router /api/v1/auth/check-username [get]
func (h *AuthHandlerImpl) CheckUsername(ctx *gin.Context) {
	username := ctx.Query("username")
	if username == "" {
		lib.RespondError(ctx, http.StatusBadRequest, "Username query parameter is required", nil)
		return
	}

	exists, err := h.authService.CheckUsername(ctx.Request.Context(), username)
	if err != nil {
		lib.RespondError(ctx, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	lib.RespondSuccess(ctx, http.StatusOK, "Username availability checked", gin.H{
		"exists": exists,
	})
}
