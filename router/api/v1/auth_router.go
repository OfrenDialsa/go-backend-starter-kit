package v1

import (
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(rg *gin.RouterGroup, h handler.AuthHandler, mw middleware.Middleware) {
	auth := rg.Group("/auth")
	{
		auth.POST("/check-availability", mw.RateLimit(30, time.Minute), h.CheckAvailability)
		auth.POST("/register", mw.RateLimit(10, time.Minute), h.Register)
		auth.POST("/resend-verification", mw.RateLimit(10, time.Minute), h.ResendVerification)
		auth.GET("/verify-email", mw.RateLimit(10, time.Minute), h.VerifyEmail)
		auth.POST("/login", mw.RateLimit(10, time.Minute), h.Login)
		auth.POST("/forgot-password", mw.RateLimit(3, 15*time.Minute), h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", mw.Validate(), h.Logout)
	}
}
