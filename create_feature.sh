#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: ./create_feature.sh <feature_name>"
    exit 1
fi

FEATURE_NAME=$1
PASCAL_FEATURE_NAME=$(echo "$FEATURE_NAME" | awk -F'_' '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)} }1' OFS='')
CAMEL_FEATURE_NAME="$(tr '[:upper:]' '[:lower:]' <<< ${PASCAL_FEATURE_NAME:0:1})${PASCAL_FEATURE_NAME:1}"

echo "~ Creating feature: $FEATURE_NAME"

# --- 1. REPOSITORY ---
cat > internal/repository/${FEATURE_NAME}_repository.go << EOF
package repository

type ${PASCAL_FEATURE_NAME}Repository interface {
    // add your repository here
}
EOF

cat > internal/repository/${FEATURE_NAME}_repository_impl.go << EOF
package repository

import (
    "github/OfrenDialsa/go-gin-starter/database"
)

type ${CAMEL_FEATURE_NAME}RepositoryImpl struct {
    db *database.WrapDB
}

func New${PASCAL_FEATURE_NAME}Repository(db *database.WrapDB) ${PASCAL_FEATURE_NAME}Repository {
    return &${CAMEL_FEATURE_NAME}RepositoryImpl{db: db}
}
EOF

# --- 2. SERVICE ---
cat > internal/service/${FEATURE_NAME}_service.go << EOF
package service

type ${PASCAL_FEATURE_NAME}Service interface {
    // add your service here
}
EOF

cat > internal/service/${FEATURE_NAME}_service_impl.go << EOF
package service

import (
    "github/OfrenDialsa/go-gin-starter/config"
    "github/OfrenDialsa/go-gin-starter/internal/repository"
)

type ${CAMEL_FEATURE_NAME}ServiceImpl struct {
    env  *config.EnvironmentVariable
    repo repository.${PASCAL_FEATURE_NAME}Repository
}

func New${PASCAL_FEATURE_NAME}Service(env *config.EnvironmentVariable, repo repository.${PASCAL_FEATURE_NAME}Repository) ${PASCAL_FEATURE_NAME}Service {
    return &${CAMEL_FEATURE_NAME}ServiceImpl{
        env:  env,
        repo: repo,
    }
}
EOF

# --- 3. HANDLER ---
cat > internal/handler/${FEATURE_NAME}_handler.go << EOF
package handler

import "github.com/gin-gonic/gin"

type ${PASCAL_FEATURE_NAME}Handler interface {
    // add your handler here
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
        // add your route here
        group.GET("", h.Get)
    }
}
EOF

sed -i "/type Repositories struct {/a \    ${PASCAL_FEATURE_NAME} repository.${PASCAL_FEATURE_NAME}Repository" cmd/api/repositories.go
sed -i "/return Repositories{/a \        ${PASCAL_FEATURE_NAME}: repository.New${PASCAL_FEATURE_NAME}Repository(db)," cmd/api/repositories.go

sed -i "/type Services struct {/a \    ${PASCAL_FEATURE_NAME} service.${PASCAL_FEATURE_NAME}Service" cmd/api/services.go
sed -i "/return Services{/a \        ${PASCAL_FEATURE_NAME}: service.New${PASCAL_FEATURE_NAME}Service(env, r.${PASCAL_FEATURE_NAME})," cmd/api/services.go

sed -i "/type Handlers struct {/a \    ${PASCAL_FEATURE_NAME} handler.${PASCAL_FEATURE_NAME}Handler" cmd/api/handlers.go
sed -i "/return Handlers{/a \        ${PASCAL_FEATURE_NAME}: handler.New${PASCAL_FEATURE_NAME}Handler(s.${PASCAL_FEATURE_NAME})," cmd/api/handlers.go

sed -i "/UserHandler handler.UserHandler/a \    ${PASCAL_FEATURE_NAME}Handler handler.${PASCAL_FEATURE_NAME}Handler" router/router.go
sed -i "/features.UserRoutes(v1, h.UserHandler, h.Middleware)/a \            features.${PASCAL_FEATURE_NAME}Routes(v1, h.${PASCAL_FEATURE_NAME}Handler, h.Middleware)" router/router.go
sed -i "/UserHandler: handlers.User,/a \        ${PASCAL_FEATURE_NAME}Handler: handlers.${PASCAL_FEATURE_NAME}," cmd/api/init.go

echo "---"
echo "✅ Files created for feature: $PASCAL_FEATURE_NAME"