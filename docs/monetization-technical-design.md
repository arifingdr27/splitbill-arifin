# Monetization Technical Design

| | |
|---|---|
| **Document type** | Technical Design Document (TDD) |
| **Status** | Draft — menunggu approval keputusan §13 |
| **Product** | Splitbill (BE + FE) |
| **Model** | Freemium kuota OCR bulanan + paket kredit (pay-per-use) |
| **Scope** | Identity ringan, kuota server-side, payment QRIS, gate di OCR |

**Related codebases**

- Backend: `splitbill-arifin` (Go / Fiber)
- Frontend: `splitbill-frontend` (React / Vite / Redux)

---

## 1. Latar belakang

Aplikasi saat ini:

- **BE:** API stateless `POST /api/v2` — upload gambar struk → ekstraksi via Gemini → JSON.
- **FE:** SPA 5 langkah (upload → edit → teman → assign → hasil). Hanya OCR yang memanggil BE; sisanya client-side.
- **Belum ada:** database, auth, payment, subscription, usage tracking.

Unit yang bermodal dan bernilai monetisasi adalah **OCR** (biaya Gemini). Titik enforce paling natural: sebelum pemanggilan Gemini di `POST /api/v2`.

### Model bisnis yang diadopsi

| Aturan | Nilai MVP |
|--------|-----------|
| Free quota | 5 OCR / user / **kalender bulan** (WIB, UTC+7) |
| Konsumsi | +1 hanya jika OCR **sukses** (HTTP 200) |
| Gagal OCR (422/503) | **tidak** mengurangi kuota |
| Paket | `PACK_20` = 20 kredit, harga **Rp15.000** |
| Prioritas potong | **kredit paket dulu**, lalu free monthly |
| Kredit | tidak expired di MVP |
| Subscription | tidak ada di MVP |

**Bukan** daily limit 2×/hari — frekuensi pakai split bill biasanya sporadis, sehingga limit harian jarang kepicu dan konversi lemah.

---

## 2. Prinsip desain

1. **Choke-point tunggal:** hanya sukses `POST /api/v2` yang mengonsumsi kuota.
2. **Kuota di server**, bukan `localStorage`.
3. **OCR tidak dijalankan** jika kuota habis (cek sebelum Gemini).
4. **Clean Architecture BE tetap:** port baru + adapter baru; middleware di HTTP layer.
5. **FE tetap SPA:** auth ringan + paywall di upload; flow split 5 langkah tidak berubah.

---

## 3. Arsitektur target (MVP)

```
[FE React]
   │ Bearer JWT
   ▼
[BE Fiber]
   ├─ Auth middleware
   ├─ Quota check (sebelum Extract)
   ├─ SplitbillService (existing Gemini)
   ├─ BillingService (checkout, apply kredit)
   └─ Webhook Midtrans/Xendit
         │
         ▼
   [PostgreSQL]
```

### Stack tambahan

| Komponen | Pilihan MVP |
|----------|-------------|
| Database | PostgreSQL |
| Auth | Google OAuth + JWT (HS256) |
| Payment | Midtrans atau Xendit (QRIS) |
| Rate limit | In-memory / per-IP dulu; Redis opsional fase berikutnya |

---

## 4. Aturan debit (kontrak runtime)

```
remaining = credit_balance + free_remaining_this_month
if remaining <= 0 → reject 402
else → jalankan OCR
on success → debit 1 (credit dulu, else free)
on failure → no debit
```

---

## 5. Database schema

### 5.1 `users`

| Column | Type | Keterangan |
|--------|------|------------|
| id | UUID PK | |
| email | TEXT UNIQUE | |
| name | TEXT | |
| google_sub | TEXT UNIQUE NULL | |
| credit_balance | INT | cache; default 0 |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### 5.2 `usage_monthly`

| Column | Type | Keterangan |
|--------|------|------------|
| id | UUID PK | |
| user_id | UUID FK → users | |
| period_ym | CHAR(7) | contoh `2026-09` (bulan WIB) |
| free_used | INT | default 0 |
| free_limit | INT | default 5 |
| UNIQUE(user_id, period_ym) | | |

### 5.3 `credit_ledger`

Append-only untuk audit.

| Column | Type | Keterangan |
|--------|------|------------|
| id | UUID PK | |
| user_id | UUID FK → users | |
| delta | INT | +20 beli, -1 pakai |
| reason | TEXT | `purchase`, `ocr`, `admin_grant` |
| ref_id | TEXT NULL | payment_id / extract event id |
| created_at | TIMESTAMPTZ | |

Balance kanonik dapat diverifikasi sebagai `SUM(delta)`; kolom `users.credit_balance` di-update dalam transaksi yang sama.

### 5.4 `payments`

| Column | Type | Keterangan |
|--------|------|------------|
| id | UUID PK | |
| user_id | UUID FK → users | |
| provider | TEXT | `midtrans` / `xendit` |
| provider_order_id | TEXT UNIQUE | |
| product_code | TEXT | `PACK_20` |
| amount_idr | INT | 15000 |
| status | TEXT | `pending`, `paid`, `failed`, `expired` |
| raw_webhook | JSONB NULL | |
| created_at | TIMESTAMPTZ | |
| paid_at | TIMESTAMPTZ NULL | |

### 5.5 `ocr_usage_events` (opsional, disarankan)

| Column | Type | Keterangan |
|--------|------|------------|
| id | UUID PK | |
| user_id | UUID FK → users | |
| source | TEXT | `credit` \| `free` |
| status | TEXT | `success` \| `failed_no_charge` |
| created_at | TIMESTAMPTZ | |

### 5.6 Index penting

- `usage_monthly (user_id, period_ym)` — unique sudah mencakup
- `credit_ledger (user_id, created_at)`
- `payments (provider_order_id)` — unique
- `ocr_usage_events (user_id, created_at)`

---

## 6. Auth

### 6.1 Flow

1. FE: tombol “Lanjut dengan Google”.
2. BE: `POST /api/v1/auth/google` dengan `id_token`.
3. BE verifikasi token Google → upsert `users` → terbitkan **JWT** (expiry 30 hari MVP).
4. FE simpan JWT.
5. Request OCR mengirim `Authorization: Bearer <jwt>`.

### 6.2 Policy MVP

- **OCR wajib login** dari request pertama (paling bersih untuk kuota & anti-abuse).
- Anonymous free tier **tidak** dipakai di v1.

### 6.3 JWT claims (minimal)

```json
{
  "sub": "<user_uuid>",
  "email": "user@example.com",
  "exp": 1234567890
}
```

---

## 7. API design

- OCR tetap di **`/api/v2`** (kompatibilitas existing).
- Modul baru di **`/api/v1/*`**.

### 7.1 Auth

| Method | Path | Auth | Fungsi |
|--------|------|------|--------|
| POST | `/api/v1/auth/google` | No | Tukar Google `id_token` → JWT + profil user |

### 7.2 Quota

| Method | Path | Auth | Fungsi |
|--------|------|------|--------|
| GET | `/api/v1/me/quota` | Yes | Sisa free + kredit + total |

**Contoh response**

```json
{
  "period": "2026-09",
  "free_limit": 5,
  "free_used": 2,
  "free_remaining": 3,
  "credit_balance": 0,
  "total_remaining": 3
}
```

### 7.3 Billing

| Method | Path | Auth | Fungsi |
|--------|------|------|--------|
| GET | `/api/v1/products` | Yes | Daftar paket |
| POST | `/api/v1/payments/checkout` | Yes | Buat order + URL/QR bayar |
| POST | `/api/v1/payments/webhook` | Signature provider | Update status + grant kredit |
| GET | `/api/v1/payments/{id}` | Yes | Poll status (opsional) |

**Checkout request**

```json
{
  "product_code": "PACK_20"
}
```

**Checkout response**

```json
{
  "payment_id": "...",
  "status": "pending",
  "redirect_url": "https://...",
  "qr_string": "..."
}
```

### 7.4 OCR (existing, diubah)

| Method | Path | Auth | Perubahan |
|--------|------|------|-----------|
| POST | `/api/v2` | **Yes** | Cek kuota → extract → debit jika sukses |

**Error kuota habis** — HTTP **402 Payment Required**

```json
{
  "status": "quota_exceeded",
  "data": "Free quota and credits are exhausted",
  "quota": {
    "total_remaining": 0
  }
}
```

**Opsional** pada response sukses: field `_meta.quota_remaining_after`, atau FE re-fetch `GET /me/quota`.

### 7.5 Admin grant (opsional first money)

| Method | Path | Auth | Fungsi |
|--------|------|------|--------|
| POST | `/api/v1/admin/grants` | Admin secret | Grant `+N` kredit manual |

---

## 8. Flow runtime

### 8.1 Upload OCR

```
FE (ada JWT?)
  no  → tampilkan login
  yes → (opsional) GET /me/quota untuk UI
      → POST /api/v2 + Bearer

BE:
  validate JWT
  lock user / usage row (SELECT … FOR UPDATE)
  if credit + free <= 0 → 402
  panggil Gemini
  if fail → no debit, return error
  if ok  → debit (credit dulu), insert event, return JSON
```

Debit **hanya setelah** OCR sukses. Race condition dicegah dengan row lock sebelum cek kuota.

### 8.2 Beli paket

```
FE → POST /payments/checkout
BE → insert payments(pending) → create order di payment provider
FE → user bayar QRIS
Provider → POST /payments/webhook
BE → verify signature
   → jika paid & belum processed (idempotent):
        update payment → paid
        insert credit_ledger +20
        update users.credit_balance
FE → poll status / refresh quota → lanjut upload
```

---

## 9. Perubahan frontend

| Area | Perubahan |
|------|-----------|
| Auth | Modal/halaman Google login; simpan JWT |
| `receiptApi.js` | Header `Authorization`; handle 401 & 402 |
| Upload `/` | Tampilkan sisa kuota; 402 → modal beli paket |
| Billing | Pilih `PACK_20` → redirect/QR → sukses → refresh quota |
| Router | Opsional: `/login`, `/billing` |
| Guard | Belum login → login sebelum upload |

Flow 5 langkah split **tidak diubah**.

---

## 10. Perubahan backend (struktur modul)

```
internal/
  domain/          # User, Quota, Payment, Product
  port/            # UserRepo, UsageRepo, PaymentRepo, PaymentProvider, TokenVerifier
  service/         # AuthService, QuotaService, BillingService
                   # SplitbillService: inject QuotaService
  adapter/
    http/          # auth, billing, webhook, auth middleware
    postgres/      # repositories
    google/        # verify id_token
    midtrans/      # checkout + webhook verify (atau xendit/)
    gemini/        # existing
    storage/       # existing
```

### Config baru (nama env)

| Env | Keterangan |
|-----|------------|
| `DATABASE_URL` | PostgreSQL DSN |
| `JWT_SECRET` | Signing key |
| `GOOGLE_CLIENT_ID` | Verifikasi id_token |
| `MIDTRANS_*` / `XENDIT_*` | Kredensial payment |
| `FREE_OCR_LIMIT` | Default `5` |
| `ADMIN_GRANT_SECRET` | Opsional endpoint grant |

---

## 11. Anti-abuse

| Ancaman | Mitigasi |
|---------|----------|
| Clear browser / ganti device | Kuota terikat `user_id` |
| Multi-account Google | Rate limit IP (signup/OCR per jam); review manual awal |
| Double webhook | Idempotent: grant hanya pada transisi `pending → paid` |
| Bypass FE | Enforce di BE sebelum Gemini |
| Spam request gagal | Rate limit per user (mis. 10 req/menit) meski tidak debit |
| Replay JWT | Expiry + HTTPS; refresh token di fase berikutnya |

---

## 12. Observability

Log terstruktur (Logrus existing), sertakan bila relevan:

- `user_id`
- `quota_remaining`
- `debit_source` (`credit` \| `free`)
- `payment_id`
- latency OCR

Metrik kasar awal: OCR/hari, checkout → paid conversion.

---

## 13. Keputusan yang perlu di-approve

Sebelum implementasi:

1. **Auth:** Google OAuth saja untuk MVP?
2. **OCR wajib login** dari request pertama?
3. **Payment:** Midtrans atau Xendit? (atau Fase 1 dulu tanpa payment?)
4. **Paket:** satu SKU `PACK_20 @ Rp15.000` cukup?
5. **Free limit:** 5 / bulan (bukan 2 / hari)?
6. **DB:** PostgreSQL OK?

---

## 14. Urutan implementasi

| Fase | Isi | Hasil |
|------|-----|--------|
| **0** | Postgres + migrasi + Wire DB | Fondasi |
| **1** | Google login, JWT, `usage_monthly`, enforce di `/api/v2`, UI kuota | Menahan biaya Gemini; belum revenue |
| **2** | `payments` + `credit_ledger`, checkout, webhook, modal beli paket | **First money** |
| **3** | Idempotency ketat, admin grant, rate limit, meta quota, monitoring | Penguatan |
| **4** | History bill, share link, subscription, device fingerprint | Belakangan |

---

## 15. Estimasi revenue (referensi, bukan proyeksi agresif)

Asumsi: konversi ~3% user aktif membeli 1× paket Rp15.000 / bulan.

| User aktif / bulan | Pembayar (~3%) | Revenue kasar |
|--------------------|----------------|---------------|
| 100 | 3 | Rp45.000 |
| 500 | 15 | Rp225.000 |
| 1.000 | 30 | Rp450.000 |

Biaya Gemini + hosting harus tetap di bawah effective price per scan agar margin positif.

---

## 16. Yang tidak masuk scope dokumen ini

- Implementasi code / migrasi SQL final
- UI/UX pixel-level
- Legal (ToS, kebijakan refund)
- Subscription & fitur premium non-OCR (history, share) — fase belakangan

---

## Riwayat dokumen

| Tanggal | Perubahan |
|---------|-----------|
| 2026-09-12 | Draft awal TDD monetisasi OCR quota + credit pack |
