# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/splitbill-api ./cmd/api

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/splitbill-api /app/splitbill-api

RUN mkdir -p /app/storage/public/images /app/storage/logs/general_log

EXPOSE 3000

CMD ["/app/splitbill-api"]
