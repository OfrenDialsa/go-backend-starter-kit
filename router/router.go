package router

import (
	"github/OfrenDialsa/go-gin-starter/config"
	_ "github/OfrenDialsa/go-gin-starter/docs"
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/internal/metrics"
	"github/OfrenDialsa/go-gin-starter/middleware"
	apiV1 "github/OfrenDialsa/go-gin-starter/router/api/v1"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Env         *config.EnvironmentVariable
	Middleware  middleware.Middleware
	AuthHandler handler.AuthHandler
	UserHandler handler.UserHandler
}

func NewRouter(env *config.EnvironmentVariable, h Handler) *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(metrics.PrometheusMiddleware())

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
		{
			apiV1.AuthRoutes(v1, h.AuthHandler, h.Middleware)
			apiV1.UserRoutes(v1, h.UserHandler, h.Middleware)
		}

		PrometheusRouter(env, router)

		if env.App.Mode == "dev" {
			setupSwagger(base, env)
			pprof.Register(router)
		}
	}
	return router
}
