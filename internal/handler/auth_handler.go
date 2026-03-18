package handler

import "github.com/gin-gonic/gin"

type AuthHandler interface {
	Register(ctx *gin.Context)
	VerifyEmail(ctx *gin.Context)
	Login(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Logout(ctx *gin.Context)
	ForgotPassword(ctx *gin.Context)
	ResetPassword(ctx *gin.Context)
	CheckEmail(ctx *gin.Context)
	CheckUsername(ctx *gin.Context)
}
