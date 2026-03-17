package features

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
		users.PUT("/me", h.UpdateProfile)
		users.PUT("/me/password", h.ChangePassword)
		users.DELETE("/me/avatar", h.DeleteAvatar)
		users.DELETE("/me", h.GetMe)
	}
}
