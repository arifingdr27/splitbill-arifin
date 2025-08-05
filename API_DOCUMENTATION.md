# Splitbill API Documentation

## Overview
Splitbill API adalah layanan yang memungkinkan ekstraksi informasi dari gambar struk belanja menggunakan teknologi OCR (Optical Character Recognition) dan AI. API ini dapat mengidentifikasi item-item yang dibeli, informasi toko, total harga, pajak, dan detail transaksi lainnya.

## Quick Start

### Prerequisites
- Go 1.23 atau lebih baru
- API Key untuk Google Gemini AI
- Firebase Project untuk storage (opsional)

### Installation
1. Clone repository ini
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Setup environment variables (buat file `.env`):
   ```
   GEMINI_API_KEY=your_gemini_api_key
   BUCKET_STORAGE=VM # atau FIREBASE
   ```
4. Jalankan aplikasi:
   ```bash
   go run main.go
   ```

## API Documentation

### Base URL
```
http://localhost:3000
```

### Swagger Documentation
Dokumentasi interactive Swagger tersedia di:
```
http://localhost:3000/swagger/index.html
```

### Endpoints

#### POST /api/v2
Extract splitbill information from receipt image

**Request:**
- Method: `POST`
- Content-Type: `multipart/form-data`
- Parameters:
  - `image` (file, required): Receipt image file (jpg, jpeg, png)

**Response Success (202):**
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
  }
}
```

**Response Error (406):**
```json
{
  "data": "",
  "status": "Error uploading image"
}
```

## Features

- **OCR Processing**: Menggunakan Google Gemini AI untuk membaca teks dari gambar struk
- **Smart Extraction**: Mengidentifikasi dan mengekstrak informasi terstruktur dari struk
- **Flexible Storage**: Mendukung penyimpanan gambar ke VM lokal atau Firebase Storage
- **Detailed Response**: Memberikan informasi lengkap termasuk item, toko, pajak, dan transaksi
- **Error Handling**: Comprehensive error handling dan logging

## Testing

### Using cURL
```bash
curl -X POST http://localhost:3000/api/v2 \
  -H "Content-Type: multipart/form-data" \
  -F "image=@/path/to/receipt.jpg"
```

### Using Swagger UI
1. Buka http://localhost:3000/swagger/index.html
2. Klik pada endpoint POST /
3. Klik "Try it out"
4. Upload file gambar struk
5. Klik "Execute"

## Supported Image Formats
- JPEG (.jpg, .jpeg)
- PNG (.png)
- Ukuran file maksimal: sesuai konfigurasi server

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GEMINI_API_KEY` | API key untuk Google Gemini AI | Required |
| `BUCKET_STORAGE` | Storage type (VM/FIREBASE) | VM |
| `FIREBASE_PROJECT_ID` | Firebase project ID (jika menggunakan Firebase) | - |

## Error Codes

| Status Code | Description |
|-------------|-------------|
| 202 | Success - Receipt processed successfully |
| 406 | Not Acceptable - Failed to process receipt |

## Development

### Generate Swagger Documentation
Setelah mengubah anotasi Swagger di kode:
```bash
swag init
```

### Project Structure
```
.
├── controllers/        # API controllers
├── services/          # Business logic
├── models/           # Data models untuk Swagger
├── docs/             # Generated Swagger documentation  
├── config/           # Configuration files
├── helpers/          # Utility functions
├── routes/           # Route definitions
└── storage/          # File storage
```

## License
Apache 2.0

## Support
Untuk bantuan teknis, silakan hubungi support@swagger.io
