package features

import (
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(rg *gin.RouterGroup, h handler.AuthHandler, mw middleware.Middleware) {
	auth := rg.Group("/auth")
	{
		auth.GET("/check-email", mw.RateLimit(5, time.Minute), h.CheckEmail)
		auth.GET("/check-username", mw.RateLimit(5, time.Minute), h.CheckUsername)
		auth.POST("/register", mw.RateLimit(10, time.Minute), h.Register)
		auth.POST("/login", mw.RateLimit(10, time.Minute), h.Login)
		auth.POST("/forgot-password", mw.RateLimit(3, 15*time.Minute), h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", mw.Validate(), h.Logout)
	}
}
