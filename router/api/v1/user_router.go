package v1

import (
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup, h handler.UserHandler, mw middleware.Middleware) {
	users := rg.Group("/users")
	users.Use(mw.Validate())
	{
		users.GET("/me", h.GetMe)
		users.PUT("/me", mw.EmailVerified(), h.UpdateProfile)
		users.PUT("/me/password", h.ChangePassword)
		users.DELETE("/me/avatar", h.DeleteAvatar)
		users.DELETE("/me", h.DeleteAccount)
	}
}
