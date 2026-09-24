# Splitbill API Documentation

## Overview

Splitbill API mengekstrak informasi dari gambar struk belanja menggunakan OCR + AI (item, toko, totals, `fees[]`, transaksi).

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

### Totals contract (`fees[]` + legacy)

- **Source of truth:** `totals.fees[]` — setiap biaya non-item (tax, service_charge, tip, fee, other).
- **Legacy (tetap diisi dari agregasi BE):**
  - `totals.tax.amount` / `total_tax` = sum `fees` where `type=tax`
  - `totals.tax.name` = nama fee tax terbesar (atau pertama)
  - `totals.service_charge` dan `totals.tax.service_charge` = sum `type=service_charge`
- Angka: plain decimal `"18564.00"` (tanpa pemisah ribuan). Field kosong → `null`.
- Invariant: `sum(items.total) ≈ subtotal`; `subtotal - discount + sum(fees.amount) ≈ total`.

### FE migration notes

- Migrate `getGlobalFees` ke agregasi `sum(fees[])` proporsional per `type`.
- Multi-tax / tip / packing hanya lengkap di `fees[]`; legacy tax adalah **jumlah** semua tax.
- Sampai FE migrate, legacy fields tetap diisi BE — jangan andalkan hanya `tax` tunggal untuk detail per baris.
- Breaking soft: field opsional sekarang `null` (bukan `""`).

**Success example A — 1 pajak + service:**

```json
{
  "items": [
    {"name": "Nasi", "price": "50000.00", "quantity": "2", "total": "100000.00"},
    {"name": "Ayam", "price": "41000.00", "quantity": "2", "total": "82000.00"}
  ],
  "store_information": {
    "store_name": "Rumah Makan Padang",
    "address": "Jl. Contoh",
    "email": null,
    "npwp": null,
    "phone_number": null
  },
  "totals": {
    "subtotal": "182000.00",
    "discount": "0.00",
    "fees": [
      {"type": "service_charge", "name": "Service Charge", "amount": "3640.00", "rate": null},
      {"type": "tax", "name": "PB1", "amount": "18564.00", "rate": null}
    ],
    "tax": {
      "name": "PB1",
      "amount": "18564.00",
      "total_tax": "18564.00",
      "dpp": null,
      "service_charge": "3640.00"
    },
    "service_charge": "3640.00",
    "total": "204204.00",
    "payment": "204204.00",
    "change": null
  },
  "transaction_information": {"date": "24/09/2026", "time": null, "transaction_id": null},
  "currency": {"code": "IDR", "symbol": "Rp", "name": "Indonesian Rupiah", "confidence": "high"},
  "language": {"code": "id", "name": "Indonesian", "confidence": "high"}
}
```

**Example B — multi pajak + service:**

```json
{
  "totals": {
    "subtotal": "100000.00",
    "discount": "0.00",
    "fees": [
      {"type": "service_charge", "name": "Service 5%", "amount": "5000.00", "rate": "5"},
      {"type": "tax", "name": "PB1", "amount": "10000.00", "rate": "10"},
      {"type": "tax", "name": "PPN", "amount": "11000.00", "rate": "11"}
    ],
    "tax": {
      "name": "PPN",
      "amount": "21000.00",
      "total_tax": "21000.00",
      "dpp": null,
      "service_charge": "5000.00"
    },
    "service_charge": "5000.00",
    "total": "126000.00",
    "payment": "126000.00",
    "change": null
  }
}
```

**Example C — + tip:**

```json
{
  "totals": {
    "subtotal": "182000.00",
    "discount": "0.00",
    "fees": [
      {"type": "service_charge", "name": "Service Charge", "amount": "3640.00", "rate": null},
      {"type": "tax", "name": "PB1", "amount": "18564.00", "rate": null},
      {"type": "tip", "name": "Tip", "amount": "5000.00", "rate": null}
    ],
    "tax": {
      "name": "PB1",
      "amount": "18564.00",
      "total_tax": "18564.00",
      "dpp": null,
      "service_charge": "3640.00"
    },
    "service_charge": "3640.00",
    "total": "209204.00",
    "payment": "209204.00",
    "change": null
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
cmd/api → adapter/http → service → Normalize(fees) → port (storage | gemini/groq)
```

## License

Apache 2.0
