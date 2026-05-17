package middleware

import (
	"fmt"
	"github/OfrenDialsa/go-gin-starter/config"
	"github/OfrenDialsa/go-gin-starter/database"
	"github/OfrenDialsa/go-gin-starter/internal/infra/metrics"
	"github/OfrenDialsa/go-gin-starter/internal/repository"
	"github/OfrenDialsa/go-gin-starter/lib"
	"github/OfrenDialsa/go-gin-starter/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type MiddlewareImpl struct {
	env         *config.EnvironmentVariable
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	store       sync.Map
}

func NewMiddleware(
	env *config.EnvironmentVariable,
	db *database.WrapDB,
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
			if !utils.ContainsRole(lib.Role(session.Role), roles) {
				lib.RespondError(c, lib.ErrForbidden)
				c.Abort()
				return
			}
		}

		c.Set("is_verified", session.EmailVerifiedAt != nil)
		c.Set("user", claims)

		c.Next()
	}
}

func (m *MiddlewareImpl) EmailVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		isVerified, exists := c.Get("is_verified")
		if !exists || isVerified == false {
			lib.RespondError(c, lib.ErrEmailNotVerified)
			c.Abort()
			return
		}
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

		now := time.Now()
		windowSec := window.Seconds()

		currWindowStart := now.Truncate(window).Unix()

		actual, _ := m.store.LoadOrStore(key, &utils.CounterWindow{
			CurrWindow: currWindowStart,
		})

		w := actual.(*utils.CounterWindow)

		w.Mu.Lock()

		if currWindowStart > w.CurrWindow {
			if currWindowStart-w.CurrWindow == int64(window.Seconds()) {
				w.PrevCount = w.CurrCount
			} else {
				w.PrevCount = 0
			}
			w.LastWindow = w.CurrWindow
			w.CurrWindow = currWindowStart
			w.CurrCount = 0
		}

		timePassed := now.Sub(time.Unix(w.CurrWindow, 0)).Seconds()
		weight := (windowSec - timePassed) / windowSec

		count := int(float64(w.PrevCount)*weight) + w.CurrCount

		if count >= limit {
			w.Mu.Unlock()

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			lib.RespondError(c, lib.ErrTooManyRequest)
			c.Abort()
			return
		}

		w.CurrCount++
		w.Mu.Unlock()

		c.Next()
	}
}

func (m *MiddlewareImpl) CleanRateLimit(interval time.Duration, maxAge time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			now := time.Now().Unix()

			m.store.Range(func(key, value interface{}) bool {
				w := value.(*utils.CounterWindow)
				w.Mu.Lock()

				if now-w.CurrWindow > int64(maxAge.Seconds()) {
					w.Mu.Unlock()
					m.store.Delete(key)
				} else {
					w.Mu.Unlock()
				}
				return true
			})
		}
	}()
}

func (m *MiddlewareImpl) Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		endpoint := c.FullPath()
		if endpoint == "/metrics" || strings.HasPrefix(endpoint, "/swagger") {
			c.Next()
			return
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		metrics.HTTPRequests.WithLabelValues(method, endpoint, status).Inc()
		metrics.HTTPDuration.WithLabelValues(method, endpoint).
			Observe(time.Since(start).Seconds())
	}
}
