APP_BIN := bin/splitbill-api
CMD_PKG := ./cmd/api
GO_IMAGE := golang:1.23-alpine

.PHONY: dirs build up down clean

dirs:
	mkdir -p bin storage/public/images storage/logs/general_log

build: dirs
	docker run --rm \
		-e CGO_ENABLED=0 \
		-e GOOS=linux \
		-e GOTOOLCHAIN=local \
		-v "$(CURDIR)":/app \
		-w /app \
		-v splitbill_gomod:/go/pkg/mod \
		-v splitbill_gobuild:/root/.cache/go-build \
		$(GO_IMAGE) \
		sh -c 'apk add --no-cache git ca-certificates >/dev/null && go build -ldflags="-s -w" -o $(APP_BIN) $(CMD_PKG)'

up: build
	docker compose up -d --build --remove-orphans

down:
	docker compose down

clean:
	rm -f $(APP_BIN)
