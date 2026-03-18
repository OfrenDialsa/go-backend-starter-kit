#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <feature_name>"
    exit 1
fi

FEATURE_NAME=$1
PASCAL_FEATURE_NAME=$(echo "$FEATURE_NAME" | awk -F'_' '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)} }1' OFS='')
CAMEL_FEATURE_NAME="$(tr '[:upper:]' '[:lower:]' <<< ${PASCAL_FEATURE_NAME:0:1})${PASCAL_FEATURE_NAME:1}"

echo "~ Creating Service: $PASCAL_FEATURE_NAME"

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

# Injection
sed -i "/type Services struct {/a \    ${PASCAL_FEATURE_NAME} service.${PASCAL_FEATURE_NAME}Service" cmd/api/services.go
sed -i "/return Services{/a \        ${PASCAL_FEATURE_NAME}: service.New${PASCAL_FEATURE_NAME}Service(env, r.${PASCAL_FEATURE_NAME})," cmd/api/services.go

echo "✅ Service created."