# Splitbill API Documentation

## Overview

Splitbill API mengekstrak informasi dari gambar struk belanja menggunakan OCR + Google Gemini AI (item, toko, total, pajak, transaksi).

## Prerequisites

- Go 1.23+
- `GEMINI_API_KEY`
- Firebase credentials (hanya jika `BUCKET_STORAGE=FIREBASE`)

## Installation

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_PORT` | Listen port | `3000` |
| `APP_ENV` | Environment name | `development` |
| `CORS_ALLOW_ORIGINS` | CORS origins | `*` |
| `GEMINI_API_KEY` | Google Gemini API key | required |
| `GEMINI_MODEL` | Gemini model | `gemini-3.6-flash` |
| `BUCKET_STORAGE` | `VM` or `FIREBASE` | `VM` |
| `STORAGE_LOCAL_PATH` | Local upload path | `./storage/public` |
| `FIREBASE_SERVICE_ACCOUNT_KEY_PATH` | Service account JSON | `./storage/firebase-adminsdk.json` |
| `FIREBASE_STORAGE_BUCKET` | Firebase bucket name | required if FIREBASE |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_DIR` | Log directory | `./storage/logs` |

## Base URL

```
http://localhost:3000
```

Swagger: http://localhost:3000/swagger/index.html

## Endpoint

### POST /api/v2

Extract splitbill information from receipt image.

**Request:** `multipart/form-data` field `image` (jpg, jpeg, png)

**Success (200):**

```json
{
  "items": [
    {
      "name": "Nasi Goreng",
      "price": "25000.00",
      "quantity": "2",
      "total": "50000.00"
    }
  ],
  "store_information": {
    "address": "Jl. Sudirman No. 123, Jakarta",
    "email": "info@restaurant.com",
    "npwp": "12.345.678.9-012.345",
    "phone_number": "+62812345678",
    "store_name": "Restaurant ABC"
  },
  "totals": {
    "change": "5000.00",
    "discount": "0.00",
    "payment": "105000.00",
    "subtotal": "95000.00",
    "tax": {
      "amount": "5000.00",
      "service_charge": "0.00",
      "dpp": "95000.00",
      "name": "PPN",
      "total_tax": "5000.00"
    },
    "total": "100000.00"
  },
  "transaction_information": {
    "date": "02/08/2025",
    "time": "19:30",
    "transaction_id": "TXN123456789"
  },
  "currency": {
    "code": "IDR",
    "symbol": "Rp",
    "name": "Indonesian Rupiah",
    "confidence": "high"
  }
}
```

**Error (400 / 422):**

```json
{
  "data": "",
  "status": "image file is required"
}
```

## cURL

```bash
curl -X POST http://localhost:3000/api/v2 \
  -F "image=@/path/to/receipt.jpg"
```

## Docker

```bash
cp .env.example .env
docker compose up --build
```

Nginx proxy: http://localhost:8031

## Architecture

```
cmd/api → adapter/http → service → port (storage | gemini)
```

Stateless: no database. Images stored to VM disk or Firebase Storage.

## License

Apache 2.0
