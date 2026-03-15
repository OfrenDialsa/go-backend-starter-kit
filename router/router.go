package router

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/docs"
	_ "github/OfrenDialsa/go-gin-starter/docs"
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	Env         *config.EnvironmentVariable
	Middleware  middleware.Middleware
	AuthHandler handler.AuthHandler
	UserHandler handler.UserHandler
}

func NewRouter(env *config.EnvironmentVariable, handler Handler) *gin.Engine {

	router := gin.Default()
	router.Use(cors.Default())

	base := router.Group("/")
	{
		base.GET("", func(ctx *gin.Context) {
			ctx.JSON(200, "Go gin starter template by nerodev")
		})

		base.GET("/health", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{
				"success": true,
				"message": "Service is healthy",
				"data": gin.H{
					"status":   "ok",
					"database": "connected",
					"version":  "1.0.0",
				},
			})
		})

		v1 := base.Group("/api/v1")
		middleware := handler.Middleware
		{
			auth := v1.Group("/auth")
			{
				auth.POST("/register", handler.AuthHandler.Register)
				auth.POST("/login", handler.AuthHandler.Login)
				auth.POST("/forgot-password", handler.AuthHandler.ForgotPassword)
				auth.POST("/reset-password", handler.AuthHandler.ResetPassword)

				auth.POST("/logout", middleware.Validate(), handler.AuthHandler.Logout)
				auth.POST("/refresh", handler.AuthHandler.RefreshToken)
			}

			users := v1.Group("/users")
			{
				users.GET("/me", middleware.Validate(), handler.UserHandler.GetMe)
				users.PUT("/me", middleware.Validate(), handler.UserHandler.UpdateProfile)
				users.PUT("/me/password", middleware.Validate(), handler.UserHandler.ChangePassword)
				users.DELETE("/me", middleware.Validate(), handler.UserHandler.DeleteAccount)
			}
		}

		if env.App.Mode == "dev" {
			docs.SwaggerInfo.BasePath = env.Swagger.BasePath
			uiPath := "/swagger"

			log.Info().
				Str("host", env.Swagger.Host).
				Str("path", uiPath).
				Msgf("Swagger UI is available at http://%s%s/index.html", env.App.Host, uiPath)

			base.GET(uiPath+"/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}
	return router
}
