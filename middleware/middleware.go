package middleware

import (
	"github/OfrenDialsa/go-gin-starter/lib"

	"github.com/gin-gonic/gin"
)

type Middleware interface {
	Validate(roles ...lib.Role) gin.HandlerFunc
}
