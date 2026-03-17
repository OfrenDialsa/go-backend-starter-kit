package features

import (
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(rg *gin.RouterGroup, h handler.AuthHandler, mw middleware.Middleware) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.POST("/refresh", h.RefreshToken)

		auth.POST("/logout", mw.Validate(), h.Logout)
	}
}
