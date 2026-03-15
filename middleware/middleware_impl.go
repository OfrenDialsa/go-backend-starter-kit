package middleware

import (
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type MiddlewareImpl struct {
	env         *config.EnvironmentVariable
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewMiddleware(
	env *config.EnvironmentVariable,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) Middleware {
	return &MiddlewareImpl{
		env:         env,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (m *MiddlewareImpl) Validate(roles ...lib.Role) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			lib.RespondError(c, http.StatusUnauthorized, "Authorization header is required", nil)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			lib.RespondError(c, http.StatusUnauthorized, "Invalid authorization format", nil)
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := lib.ValidateToken(tokenString, m.env.JWT.SecretKey.Access)
		if err != nil {
			lib.RespondError(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		session, err := m.sessionRepo.GetBySessionId(c.Request.Context(), claims.SessionId)
		if err != nil || session == nil {
			lib.RespondError(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		if len(roles) > 0 {
			if !utils.ContainsRole(lib.Role(claims.Role), roles) {
				lib.RespondError(c, http.StatusForbidden, "Forbidden", nil)
				c.Abort()
				return
			}
		}

		c.Set("user", claims)

		c.Next()
	}
}
