# Splitbill API

Receipt OCR/AI service: upload a receipt image, get structured splitbill JSON via Gemini.

## Stack

- Go 1.23 + Fiber
- Clean Architecture (`cmd/` + `internal/`)
- Google Wire DI
- Storage: local VM or Firebase
- Gemini for OCR extraction

## Setup

```bash
cp .env.example .env
# set GEMINI_API_KEY (and Firebase vars if BUCKET_STORAGE=FIREBASE)

go mod tidy
go run ./cmd/api
```

Swagger: http://localhost:3000/swagger/index.html

## Docker

```bash
cp .env.example .env
docker compose up --build
```

API via nginx: http://localhost:8031

## Endpoint

`POST /api/v2` — multipart field `image` (jpg/jpeg/png)

## Project layout

```
cmd/api/                 entrypoint
internal/
  config/                typed env config
  domain/                DTOs + errors
  port/                  interfaces
  service/               use-cases
  adapter/http|storage|gemini
  di/                    Wire
pkg/logger/
docs/                    Swagger
```

Regenerate Wire: `cd internal/di && wire`
Regenerate Swagger: `swag init -g cmd/api/main.go -o docs`
