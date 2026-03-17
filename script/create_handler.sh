#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <feature_name>"
    exit 1
fi

FEATURE_NAME=$1
PASCAL_FEATURE_NAME=$(echo "$FEATURE_NAME" | awk -F'_' '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)} }1' OFS='')
CAMEL_FEATURE_NAME="$(tr '[:upper:]' '[:lower:]' <<< ${PASCAL_FEATURE_NAME:0:1})${PASCAL_FEATURE_NAME:1}"

echo "~ Creating Handler & Router: $PASCAL_FEATURE_NAME"

cat > internal/handler/${FEATURE_NAME}_handler.go << EOF
package handler

import "github.com/gin-gonic/gin"

type ${PASCAL_FEATURE_NAME}Handler interface {
    Get(ctx *gin.Context)
}
EOF

cat > internal/handler/${FEATURE_NAME}_handler_impl.go << EOF
package handler

import (
    "github/OfrenDialsa/go-gin-starter/internal/service"
    "github/OfrenDialsa/go-gin-starter/lib"
    "github.com/gin-gonic/gin"
)

type ${CAMEL_FEATURE_NAME}HandlerImpl struct {
    service service.${PASCAL_FEATURE_NAME}Service
}

func New${PASCAL_FEATURE_NAME}Handler(service service.${PASCAL_FEATURE_NAME}Service) ${PASCAL_FEATURE_NAME}Handler {
    return &${CAMEL_FEATURE_NAME}HandlerImpl{
        service: service,
    }
}

func (h *${CAMEL_FEATURE_NAME}HandlerImpl) Get(ctx *gin.Context) {
    lib.RespondSuccess(ctx, 200, "Success get $FEATURE_NAME", nil)
}
EOF

cat > router/features/${FEATURE_NAME}_router.go << EOF
package features

import (
    "github/OfrenDialsa/go-gin-starter/internal/handler"
    "github/OfrenDialsa/go-gin-starter/middleware"
    "github.com/gin-gonic/gin"
)

func ${PASCAL_FEATURE_NAME}Routes(rg *gin.RouterGroup, h handler.${PASCAL_FEATURE_NAME}Handler, mw middleware.Middleware) {
    group := rg.Group("/${FEATURE_NAME}")
    {
        group.GET("", h.Get)
    }
}
EOF

# Injection
sed -i "/type Handlers struct {/a \    ${PASCAL_FEATURE_NAME} handler.${PASCAL_FEATURE_NAME}Handler" cmd/api/handlers.go
sed -i "/return Handlers{/a \        ${PASCAL_FEATURE_NAME}: handler.New${PASCAL_FEATURE_NAME}Handler(s.${PASCAL_FEATURE_NAME})," cmd/api/handlers.go
sed -i "/UserHandler handler.UserHandler/a \    ${PASCAL_FEATURE_NAME}Handler handler.${PASCAL_FEATURE_NAME}Handler" router/router.go
sed -i "/features.UserRoutes(v1, h.UserHandler, h.Middleware)/a \            features.${PASCAL_FEATURE_NAME}Routes(v1, h.${PASCAL_FEATURE_NAME}Handler, h.Middleware)" router/router.go
sed -i "/UserHandler: handlers.User,/a \        ${PASCAL_FEATURE_NAME}Handler: handlers.${PASCAL_FEATURE_NAME}," cmd/api/init.go

echo "✅ Handler and Router created."