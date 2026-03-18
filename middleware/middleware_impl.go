package middleware

import (
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type MiddlewareImpl struct {
	env         *config.EnvironmentVariable
	redis       *redis.Client
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewMiddleware(
	env *config.EnvironmentVariable,
	db *database.WrapDB,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) Middleware {
	return &MiddlewareImpl{
		env:         env,
		redis:       db.Redis.Conn,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (m *MiddlewareImpl) Validate(roles ...lib.Role) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			lib.RespondError(c, lib.ErrUnauthorized)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			lib.RespondError(c, lib.ErrInvalidAuthorizationFormat)
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := lib.ValidateToken(tokenString, m.env.JWT.SecretKey.Access)
		if err != nil {
			lib.RespondError(c, lib.ErrInvalidAuthorizationFormat)
			c.Abort()
			return
		}

		session, err := m.sessionRepo.GetBySessionId(c.Request.Context(), claims.SessionId)
		if err != nil || session == nil {
			lib.RespondError(c, lib.ErrInvalidAuthorizationFormat)
			c.Abort()
			return
		}

		if len(roles) > 0 {
			if !utils.ContainsRole(lib.Role(claims.Role), roles) {
				lib.RespondError(c, lib.ErrForbidden)
				c.Abort()
				return
			}
		}

		c.Set("user", claims)

		c.Next()
	}
}

func (m *MiddlewareImpl) RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if user, exists := c.Get("user"); exists {
			if claims, ok := user.(*lib.JWTClaims); ok {
				identifier = claims.UserId
			}
		}

		key := "rate_limit:" + c.FullPath() + ":" + identifier

		now := time.Now().UnixNano()
		boundary := time.Now().Add(-window).UnixNano()
		pipe := m.redis.TxPipeline()
		pipe.ZRemRangeByScore(c, key, "0", fmt.Sprintf("%d", boundary))
		countRes := pipe.ZCard(c, key)

		pipe.ZAdd(c, key, redis.Z{
			Score:  float64(now),
			Member: now,
		})

		pipe.Expire(c, key, window)
		_, err := pipe.Exec(c)
		if err != nil {
			lib.RespondError(c, lib.ErrInternalServer)
			c.Abort()
			return
		}

		currentCount := countRes.Val()
		if int(currentCount) >= limit {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			lib.RespondError(c, lib.ErrToooManyRequest)
			c.Abort()
			return
		}

		c.Next()
	}
}
