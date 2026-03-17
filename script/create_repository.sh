#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <feature_name>"
    exit 1
fi

FEATURE_NAME=$1
PASCAL_FEATURE_NAME=$(echo "$FEATURE_NAME" | awk -F'_' '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)} }1' OFS='')
CAMEL_FEATURE_NAME="$(tr '[:upper:]' '[:lower:]' <<< ${PASCAL_FEATURE_NAME:0:1})${PASCAL_FEATURE_NAME:1}"

echo "~ Creating Repository: $PASCAL_FEATURE_NAME"

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

# Injection
sed -i "/type Repositories struct {/a \    ${PASCAL_FEATURE_NAME} repository.${PASCAL_FEATURE_NAME}Repository" cmd/api/repositories.go
sed -i "/return Repositories{/a \        ${PASCAL_FEATURE_NAME}: repository.New${PASCAL_FEATURE_NAME}Repository(db)," cmd/api/repositories.go

echo "✅ Repository created."