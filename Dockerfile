# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/splitbill-api ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/splitbill-api /app/splitbill-api

RUN mkdir -p /app/storage/public/images /app/storage/logs/general_log \
	&& chown -R appuser:appuser /app

USER appuser

EXPOSE 3000

CMD ["/app/splitbill-api"]
