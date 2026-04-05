#!/usr/bin/env sh
# scripts/gen-gosum.sh
# Generates go.sum using a temporary Docker container.
# Run once after cloning: sh scripts/gen-gosum.sh
set -e

echo "Generating go.sum via Docker..."
docker run --rm \
  -v "$(pwd):/app" \
  -w /app \
  golang:1.22-alpine \
  sh -c "go mod tidy && go mod download"

echo "go.sum generated. You can now run: make up"
