#!/usr/bin/env sh
set -e
cd "$(dirname "$0")"
mkdir -p bin storage/public/images storage/logs/general_log

docker run --rm \
	-e CGO_ENABLED=0 \
	-e GOOS=linux \
	-e GOTOOLCHAIN=local \
	-v "$PWD":/app \
	-w /app \
	-v splitbill_gomod:/go/pkg/mod \
	-v splitbill_gobuild:/root/.cache/go-build \
	golang:1.23-alpine \
	sh -c 'apk add --no-cache git ca-certificates >/dev/null && go build -buildvcs=false -ldflags="-s -w" -o bin/splitbill-api ./cmd/api'

docker compose up -d --build --remove-orphans
