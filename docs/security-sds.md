# Security SDS — Split Bill

| Field | Value |
|-------|-------|
| Document | Software Design Specification — Security Hardening |
| Apps | Backend `splitbill-arifin`, Frontend `splitbill-frontend` |
| Status | Draft — menunggu approval sebelum implementasi |
| Date | 2026-09-12 |
| Scope | Abuse protection, API hardening, upload safety; **bukan** rewrite fitur bisnis |

---

## 1. Purpose

Dokumen ini merancang proteksi keamanan untuk Split Bill agar aplikasi aman dipakai publik, dengan fokus:

> Mencegah satu actor jahat membuat API down, menghabiskan resource, atau membakar biaya Gemini hanya dengan request berulang.

SDS ini disusun dari audit codebase (read-only). **Implementasi code belum dilakukan** sampai dokumen di-approve dan dikerjakan per fase.

---

## 2. Current Architecture (as-is)

### 2.1 Backend (`splitbill-arifin`)

| Item | Detail |
|------|--------|
| Stack | Go 1.23 + Fiber v2 |
| Entry | `cmd/api/main.go` |
| Business endpoint | `POST /api/v2` → upload gambar → storage → Gemini OCR |
| Docs | `GET /swagger/*` (publik) |
| Auth | Tidak ada |
| Database | Tidak ada |
| Redis | Tidak ada |
| Rate limit | Tidak ada |
| Middleware | `recover`, CORS, request logger |

**Request flow:**

```
HTTP → Fiber (recover, CORS, logger)
     → SplitbillHandler.Extract
     → SplitbillService.ExtractReceipt
         → StorageUploader.Upload (VM | Firebase)
         → ReceiptExtractor.Extract (Gemini, max 2 attempts)
     → JSON response
```

### 2.2 Frontend (`splitbill-frontend`)

| Item | Detail |
|------|--------|
| Stack | React 19 + Vite 6 + Redux Toolkit |
| API call | `axios.post(VITE_API_URL)` multipart field `image` |
| Auth headers | Tidak ada |
| Token storage | Tidak ada |
| Split logic | 100% client-side (Redux) |
| Deploy | nginx static SPA |

### 2.3 Endpoint inventory

| Method | Path | Auth | Cost |
|--------|------|------|------|
| `POST` | `/api/v2` | None | High (Gemini + storage + possible image decode) |
| `GET` | `/swagger/*` | None | Low (discovery aid) |

Tidak ada login, health, admin, atau endpoint CRUD user.

---

## 3. Threat Model

### 3.1 Primary question

**Apakah satu user jahat bisa membuat aplikasi down / mahal dengan request berulang?**

**Jawaban as-is: Ya.** `POST /api/v2` publik, tanpa rate limit, setiap request memanggil Gemini (hingga 2×) dan menulis storage.

### 3.2 In-scope threats

| Threat | Relevansi |
|--------|-----------|
| Infinite / rapid requests | Tinggi |
| Concurrent flooding | Tinggi |
| Expensive endpoint abuse (Gemini) | Tinggi |
| Storage spam / disk fill | Tinggi |
| Oversized payload | Sedang (Fiber default ~4 MiB) |
| CORS abuse dari origin asing | Tinggi (default `*`) |
| Weak upload type validation | Sedang |
| Error information leakage | Sedang |
| Public Swagger discovery | Sedang |
| Application-layer flooding | Tinggi |
| Prompt injection → OCR integrity | Sedang (bukan RCE) |

### 3.3 Out-of-scope / N/A (saat ini)

| Threat | Alasan |
|--------|--------|
| SQL Injection | Tidak ada database |
| Login brute force / credential stuffing | Tidak ada login |
| IDOR / BOLA antar user | Tidak ada resource per-user di server |
| Session/JWT abuse | Tidak ada session/JWT |
| Command injection | Tidak ada `os/exec` |
| Classic XSS → token theft | Tidak ada token; React text escaping |

---

## 4. Findings Summary (audit)

| ID | Severity | Finding | Evidence |
|----|----------|---------|----------|
| V1 | Critical | Endpoint AI tanpa autentikasi | `router.go` `POST /api/v2` |
| V2 | Critical | Tidak ada rate limiting | `middleware.go` — hanya recover/CORS/logger |
| V3 | Critical/High | Operasi mahal: Gemini ×2 + upload + resize | `gemini/extractor.go`, `storage/vm.go` |
| V4 | High | Storage tanpa quota/TTL | `ExtractReceipt` → `Upload` selalu |
| V5 | High | CORS default `*` | `config.go` / `.env.example` |
| V6 | High | Tidak ada Read/Write timeout & BodyLimit eksplisit | `cmd/api/main.go` |
| V7 | Medium | `writeError` mengembalikan `err.Error()` ke client | `response.go` |
| V8 | Medium | Validasi tipe: extension **atau** Content-Type | `isAllowedImage` |
| V9 | Medium | Swagger publik | `router.go` |
| V10 | Low–Medium | FE nginx tanpa security headers | `splitbill-frontend/nginx.conf` |

---

## 5. Target Security Architecture (to-be)

Disesuaikan untuk aplikasi kecil (tanpa memaksa Redis/DB dulu):

```
Browser (React SPA)
    │
    ▼
[Opsional] Cloudflare / WAF
    │  - bot challenge, volume DDoS
    ▼
Nginx reverse proxy
    │  - TLS
    │  - client_max_body_size 2m
    │  - limit_req per IP pada /api/v2
    │  - proxy timeouts
    │  - block /swagger di production
    ▼
Fiber API
    │  - BodyLimit, Read/Write timeout
    │  - API key (Phase 2) / shared secret
    │  - app-level rate limit + concurrency semaphore
    │  - magic-byte + size validation
    │  - sanitized errors
    │  - context timeout ke Gemini
    ├──► Gemini API
    └──► Storage (VM / Firebase) + retention cleanup
              │
              └── [Phase 3] Redis untuk rate/quota distributed
```

### 5.1 Peran tiap layer

| Layer | Fungsi |
|-------|--------|
| CDN/WAF | Filter traffic volume tinggi, bot dasar |
| Nginx | Rate limit IP, body size, timeout, TLS, hide Swagger |
| Fiber | Auth, validasi upload, concurrency Gemini, error sanitization |
| Redis (nanti) | Counter rate/quota multi-instance |
| Database | Belum diperlukan untuk security Phase 1–2 |

---

## 6. Security Requirements

### 6.1 Functional requirements

| ID | Requirement |
|----|-------------|
| SR-01 | `POST /api/v2` harus memiliki rate limit per IP |
| SR-02 | Request body/upload maksimal **2 MiB** |
| SR-03 | HTTP Read/Write timeout terkonfigurasi (30–60s) |
| SR-04 | Maksimal **3–5** extract Gemini concurrent secara global |
| SR-05 | Gemini call memakai `context` timeout (~25s) |
| SR-06 | CORS hanya mengizinkan origin frontend production |
| SR-07 | Error response ke client generik; detail hanya di log |
| SR-08 | Swagger tidak publik di production |
| SR-09 | File harus divalidasi sebagai JPEG/PNG nyata (magic bytes / decode) |
| SR-10 | (Phase 2) Caller harus menyertakan API key / secret |
| SR-11 | (Phase 2) Quota harian per IP/key (mis. 20 extract/hari) |
| SR-12 | (Phase 2) Retention/cleanup file storage |
| SR-13 | (Phase 2) Security headers di FE nginx |

### 6.2 Non-functional

| ID | Requirement |
|----|-------------|
| NFR-01 | Proteksi Phase 1 murah & cocok single-instance kecil |
| NFR-02 | Tidak mengubah kontrak response sukses OCR kecuali disepakati |
| NFR-03 | Tidak menambah database hanya untuk security Phase 1 |
| NFR-04 | Konfigurasi via env (bukan hardcode secret) |

### 6.3 Default policy values (usulan)

| Policy | Value | Notes |
|--------|-------|-------|
| Nginx `limit_req` | 10 req/menit/IP, burst 5 | Fokus `/api/v2` |
| App rate limit | 5 extract/menit/IP | 429 + `Retry-After` |
| Daily quota | 20 extract/IP/hari | Phase 2 |
| BodyLimit | 2 MiB | Fiber + Nginx |
| Gemini concurrency | 5 global | Semaphore |
| Gemini timeout | 25s | Per attempt; max attempts boleh diturunkan ke 1 |
| CORS | Exact FE origin(s) | Hapus default `*` di production |

---

## 7. Detailed Design

### 7.1 Rate limiting

**Nginx (edge):**

- Zone keyed by `$binary_remote_addr`
- Apply pada `location` yang mem-proxy `/api/v2`
- Response: `503`/`429` sesuai config Nginx

**Fiber (app):**

- Middleware baru di `RegisterMiddleware` atau di group `/api/v2`
- Key: client IP (`c.IP()`, pastikan `X-Forwarded-For` / `ProxyHeader` benar jika di belakang Nginx)
- Storage Phase 1: in-memory map + sliding/fixed window
- Storage Phase 3: Redis

**Behaviour:**

- Exceed → HTTP `429`
- Log field: `ip`, `path`, `reason=rate_limit`

### 7.2 Request size & timeout

**File:** `cmd/api/main.go`

```text
fiber.Config{
  AppName:      "splitbill-api",
  BodyLimit:    2 * 1024 * 1024,
  ReadTimeout:  30 * time.Second,
  WriteTimeout: 60 * time.Second,
  IdleTimeout:  60 * time.Second,
  Concurrency:  128, // atau lebih rendah
}
```

**Handler:** setelah `ReadAll`, reject jika `len(data) > maxUploadBytes` dengan error domain generik.

**Nginx:** `client_max_body_size 2m;` + `proxy_read_timeout 60s;`

### 7.3 Concurrency protection (expensive endpoint)

- Package/service-level semaphore (buffered channel size N) di sekitar `extractor.Extract`
- Jika tidak dapat slot dalam waktu singkat → `429` atau `503`
- Mencegah N concurrent Gemini + decode memenuhi RAM/CPU

### 7.4 Gemini timeout & attempts

**File:** `internal/adapter/gemini/extractor.go`

- Wrap call dengan `context.WithTimeout(ctx, 25*time.Second)`
- Pertimbangkan `maxExtractAttempts = 1` di production untuk kurangi cost abuse
- Jangan leak raw Gemini error ke HTTP response

### 7.5 Upload validation

**File:** `internal/service/splitbill.go` (+ helper baru jika perlu)

1. Cek size
2. Detect content type dari magic bytes (`http.DetectContentType` / decode)
3. Allowlist hanya `image/jpeg`, `image/png`
4. Opsional: max width/height setelah decode
5. Tolak jika extension/MIME tidak konsisten dengan isi file

### 7.6 CORS

**File:** `internal/config/config.go`

- Production: `CORS_ALLOW_ORIGINS` wajib di-set ke origin FE (comma-separated)
- Tolak startup atau warn keras jika `APP_ENV=production` dan value `*`

### 7.7 Error sanitization

**File:** `internal/adapter/http/response.go`

| Domain error | HTTP | Client message |
|--------------|------|----------------|
| `ErrImageRequired` | 400 | `image file is required` |
| `ErrInvalidImageType` | 400 | `image must be jpg, jpeg, or png` |
| `ErrStorageUnavailable` | 503 | `service temporarily unavailable` |
| `ErrExtractFailed` / lainnya | 422 | `failed to process receipt` |
| Rate limit | 429 | `too many requests` |

Detail teknis hanya ke logrus.

### 7.8 Swagger gating

**File:** `internal/adapter/http/router.go`

- Mount `/swagger/*` hanya jika `APP_ENV != production`
- Atau basic auth / IP allowlist jika masih dibutuhkan di staging

### 7.9 API key (Phase 2)

Usulan sederhana (tanpa user DB):

- Env: `API_KEYS=key1,key2` atau single `API_KEY`
- Client kirim header `X-API-Key` atau `Authorization: Bearer <key>`
- Middleware reject `401` jika missing/invalid
- FE: key **jangan** di-hardcode di public repo jika key bersifat rahasia penuh; untuk SPA publik, kombinasikan dengan rate limit + CAPTCHA (API key di SPA selalu extractable — treat sebagai soft gate)

**Catatan desain:** Untuk SPA publik murni, API key di bundle **bukan** secret kuat. Proteksi utama tetap rate limit + quota + CAPTCHA/WAF. API key berguna untuk membedakan client resmi vs script random dan untuk rotasi.

### 7.10 Usage quota (Phase 2)

- Counter harian per IP (dan per API key jika ada)
- In-memory dulu; Redis jika multi-instance
- Exceed → `429` dengan pesan quota

### 7.11 Storage retention (Phase 2)

- Cron/goroutine periodik: hapus file di `STORAGE_LOCAL_PATH/images` lebih tua dari N hari (default 7)
- Firebase: object lifecycle rule di bucket (dokumentasikan di ops)

### 7.12 Frontend controls

| Control | Layer | Wajib? |
|---------|-------|--------|
| Disable double-submit / loading state | FE | UX only |
| Client max file size check sebelum upload | FE | UX only |
| CAPTCHA widget | FE + verify BE | Phase 3 recommended |
| Security headers (CSP, X-Frame-Options, nosniff) | FE nginx | Phase 2 |
| Auth, rate limit, size, type, quota | **BE** | **Wajib** |

---

## 8. Abuse Protection Matrix

| Protection | Phase | Where |
|------------|-------|-------|
| Rate limiting per IP | 1 | Nginx + Fiber |
| Rate limiting per key/user | 2 | Fiber (+ Redis later) |
| Login brute-force | N/A | Tidak ada login |
| Request timeout | 1 | Fiber + Nginx + Gemini ctx |
| Body size limit | 1 | Fiber + Nginx + handler |
| Concurrent request limit | 1 | Semaphore pada extract |
| DB connection protection | N/A | Tidak ada DB |
| Query timeout | 1 | Diganti Gemini timeout |
| API abuse protection | 1–2 | Auth soft + rate + concurrency |
| Usage quota | 2 | Counter harian |
| Expensive endpoint protection | 1 | Semua di atas fokus `/api/v2` |
| Spam creation | 2 | Quota + retention |
| Bot / automated abuse | 3 | CAPTCHA / Cloudflare |

---

## 9. Implementation Phases

### Phase 1 — Wajib sebelum production

1. Fiber `BodyLimit`, timeouts, concurrency config
2. Rate limit middleware (in-memory) pada `/api/v2`
3. Global extract semaphore
4. Gemini context timeout
5. CORS ketat (env production)
6. Sanitize `writeError`
7. Disable Swagger di production
8. Handler reject oversized payload
9. Dokumentasikan Nginx snippet (limit_req, body size) untuk deployment

**Estimasi:** rendah (1–2 hari)  
**Risk jika ditunda:** cost Gemini & DoS tetap terbuka

### Phase 2 — Penting

1. API key middleware (soft gate)
2. Daily quota per IP/key
3. Magic-byte / decode validation
4. Storage TTL cleanup
5. FE nginx security headers
6. FE client-side max size check (UX)

**Estimasi:** rendah–sedang (2–4 hari)

### Phase 3 — Hardening

1. Redis-backed rate/quota (jika scale-out)
2. CAPTCHA / Turnstile
3. Metrics & alert (429 rate, Gemini errors, disk usage)
4. `govulncheck` di CI

**Estimasi:** sedang

### Phase 4 — Advanced

1. Full WAF / Cloudflare
2. User accounts + per-user quota (jika produk tumbuh)
3. Private storage + signed URL
4. Anomaly detection

**Estimasi:** tinggi — hanya jika skala naik

---

## 10. Files Likely to Change

### Backend (`splitbill-arifin`)

| File | Change |
|------|--------|
| `cmd/api/main.go` | BodyLimit, timeouts, concurrency |
| `internal/adapter/http/middleware.go` | Rate limit, optional API key |
| `internal/adapter/http/handler.go` | Size check, better validation hook |
| `internal/adapter/http/response.go` | Sanitize errors; 429 mapping |
| `internal/adapter/http/router.go` | Conditional Swagger |
| `internal/service/splitbill.go` | Stronger image validation |
| `internal/adapter/gemini/extractor.go` | Timeout; maybe attempts |
| `internal/config/config.go` | New env: rate limits, API key, max upload, swagger flag |
| `internal/domain/errors.go` | `ErrTooManyRequests`, `ErrPayloadTooLarge` |
| `docker-compose.yml` | Env defaults / notes |
| `.env.example` | Document new vars (tanpa secret nyata) |
| Ops: Nginx config (di server / repo ops) | limit_req, body, proxy timeout |

### Frontend (`splitbill-frontend`)

| File | Change |
|------|--------|
| `nginx.conf` | Security headers |
| `src/api/receiptApi.js` | Optional API key header; generic errors |
| `src/features/receipt/...` / upload UI | Client size limit UX; CAPTCHA later |

---

## 11. Configuration (proposed env names)

Nama saja — **jangan commit nilai secret**:

| Env | Phase | Purpose |
|-----|-------|---------|
| `CORS_ALLOW_ORIGINS` | 1 | Exact FE origins (no `*` in prod) |
| `APP_ENV` | 1 | Gate Swagger |
| `HTTP_BODY_LIMIT_BYTES` | 1 | Default 2097152 |
| `HTTP_READ_TIMEOUT_SEC` | 1 | Default 30 |
| `HTTP_WRITE_TIMEOUT_SEC` | 1 | Default 60 |
| `RATE_LIMIT_RPM` | 1 | Requests per minute per IP |
| `EXTRACT_MAX_CONCURRENT` | 1 | Global Gemini concurrency |
| `GEMINI_TIMEOUT_SEC` | 1 | Per-call timeout |
| `API_KEY` / `API_KEYS` | 2 | Soft authentication |
| `DAILY_EXTRACT_QUOTA` | 2 | Per IP/key |
| `STORAGE_RETENTION_DAYS` | 2 | Cleanup |

---

## 12. Acceptance Criteria

Phase 1 dianggap selesai jika:

1. 1 IP yang melebihi rate limit mendapat `429` (bukan terus memanggil Gemini).
2. Upload > 2 MiB ditolak sebelum/ tanpa Gemini call.
3. Lebih dari N concurrent extract mengantri/ditolak; server tetap responsif.
4. Gemini hung/lambat tidak menahan request tanpa batas (timeout).
5. `CORS_ALLOW_ORIGINS` production bukan `*`.
6. Response error tidak mengandung detail internal Gemini/storage.
7. `/swagger/*` tidak accessible di production.
8. Tidak ada regression pada happy-path OCR dengan gambar valid ≤ 2 MiB.

---

## 13. Testing Plan

| Test | Expected |
|------|----------|
| Valid JPEG/PNG ≤ 2 MiB | 200 + struktur JSON OCR |
| Missing `image` field | 400 |
| File > 2 MiB | 413 atau 400 (konsisten) |
| Non-image spoof `.jpg` | 400 setelah magic-byte check (Phase 2) |
| Burst > RPM dari 1 IP | 429 |
| Parallel > max concurrent | 429/503; no crash |
| Wrong/missing API key (Phase 2) | 401 |
| Swagger di production | 404/disabled |
| Origin asing (browser) | CORS blocked |

Load smoke (manual/script): 20 concurrent POST selama 1 menit — pastikan rate/concurrency membatasi panggilan Gemini.

---

## 14. Risks & Trade-offs

| Risk | Mitigation |
|------|------------|
| API key di SPA bisa di-extract | Utamakan rate limit + quota + WAF; key = soft gate |
| In-memory rate limit hilang saat restart / tidak share multi-instance | Phase 3 Redis; single instance OK untuk Phase 1 |
| Rate limit terlalu ketat mengganggu UX | Tunable via env; mulai longgar lalu sesuaikan |
| Menyimpan gambar tidak dipakai FE | Pertimbangkan skip persist atau TTL agresif |
| False sense of security dari validasi FE | Dokumentasikan: enforcement hanya di BE |

---

## 15. Out of Scope (dokumen ini)

- Implementasi fitur login/register user
- Migrasi ke database multi-tenant
- Redesign UI/UX split bill
- Perubahan prompt/schema Gemini kecuali terkait timeout/attempts
- Perubahan yang tidak terkait security hardening

---

## 16. Approval & Next Steps

1. Review SDS ini.
2. Approve Phase 1 scope + default policy values.
3. Implementasi bertahap (satu proteksi / PR kecil jika diinginkan).
4. Verifikasi acceptance criteria.
5. Lanjut Phase 2 setelah Phase 1 stabil di production.

**Keputusan yang perlu dikonfirmasi sebelum coding:**

- [ ] Default rate: 5–10 RPM/IP apakah OK?
- [ ] Max upload 2 MiB OK?
- [ ] Max concurrent extract 5 OK?
- [ ] Apakah API key (Phase 2) diinginkan, atau cukup rate limit dulu?
- [ ] Apakah gambar perlu tetap di-persist, atau boleh skip/TTL pendek?

---

## 17. References (codebase)

- `cmd/api/main.go` — Fiber bootstrap
- `internal/adapter/http/router.go` — routes
- `internal/adapter/http/handler.go` — `Extract`
- `internal/adapter/http/middleware.go` — CORS/logger
- `internal/adapter/http/response.go` — error JSON
- `internal/service/splitbill.go` — upload + extract orchestration
- `internal/adapter/gemini/extractor.go` — Gemini calls
- `internal/adapter/storage/vm.go` / `firebase.go` — persistence
- `internal/config/config.go` — env loading
- Frontend: `src/api/receiptApi.js`, `nginx.conf`
