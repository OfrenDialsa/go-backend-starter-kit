package router

import (
	"context"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database/postgres"
	_ "github/OfrenDialsa/go-gin-starter/docs"
	"github/OfrenDialsa/go-gin-starter/internal/handler"
	"github/OfrenDialsa/go-gin-starter/middleware"
	apiV1 "github/OfrenDialsa/go-gin-starter/router/api/v1"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Env         *config.EnvironmentVariable
	Middleware  middleware.Middleware
	AuthHandler handler.AuthHandler
	UserHandler handler.UserHandler
	DB          *postgres.WrapDB
}

func NewRouter(env *config.EnvironmentVariable, h Handler) *gin.Engine {
	r := gin.Default()
	r.ForwardedByClientIP = true
	r.SetTrustedProxies(nil)

	r.Use(cors.Default())
	r.Use(h.Middleware.Prometheus())

	base := r.Group("/")
	{
		base.GET("", func(ctx *gin.Context) {
			ctx.JSON(200, "Go gin starter kit by nerodev")
		})

		base.GET("/health", func(ctx *gin.Context) {
			dbStat := "connected"
			statCode := 200
			success := true

			if h.DB == nil || h.DB.Conn == nil {
				dbStat = "not-configured"
				statCode = 500
				success = false
			} else {
				pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
				defer cancel()

				if err := h.DB.Conn.Ping(pingCtx); err != nil {
					dbStat = "disconnected"
					statCode = 500
					success = false
				}
			}

			msg := "Service is healthy"
			if !success {
				msg = "Service is unhealthy"
			}

			ctx.JSON(statCode, gin.H{
				"success": success,
				"message": msg,
				"data": gin.H{
					"status":   "ok",
					"database": dbStat,
					"version":  "1.0.0",
				},
			})
		})

		v1 := base.Group("/api/v1")
		{
			apiV1.AuthRoutes(v1, h.AuthHandler, h.Middleware)
			apiV1.UserRoutes(v1, h.UserHandler, h.Middleware)
		}

		r.Static("/public", "./storage")

		setupPrometheus(env, r)

		if env.App.Mode == "dev" {
			setupSwagger(base, env)
			pprof.Register(r)
		}
	}
	return r
}
