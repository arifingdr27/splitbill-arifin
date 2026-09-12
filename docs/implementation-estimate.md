# Estimasi Implementasi — Security + Monetization

| Field | Value |
|-------|-------|
| Document | Implementation estimate & sequencing |
| Related | `docs/security-sds.md`, `docs/monetization-technical-design.md` |
| Apps | Backend `splitbill-arifin`, Frontend `splitbill-frontend` |
| Status | Draft |
| Date | 2026-09-12 |
| Assumption | 1 developer full-stack; fokus MVP (bukan Phase 3–4 lanjutan) |

---

## 1. Ringkasan

| Dokumen | Backend | Frontend | Total (kalender) |
|---------|---------|----------|------------------|
| **security-sds** (Phase 1–2) | ~3–5 hari | ~0.5–1 hari | **~3–5 hari** |
| **monetization** (Phase 0–2) | ~8–12 hari | ~4–6 hari | **~10–14 hari** |
| **Keduanya (MVP)** | ~10–14 hari | ~4–6 hari | **~2.5–3.5 minggu** |

Overlap: rate limit, CORS, timeout, error sanitize di SDS digabung ke fondasi monetisasi → hemat ~1–2 hari vs dikerjakan terpisah penuh.

---

## 2. Urutan praktis (hemat waktu)

1. ~~**Security Phase 1** — proteksi biaya sebelum auth/payment~~ ✅ **selesai**
2. **Monetization 0–1** — login + kuota ← *sedang dikerjakan*
3. **Monetization 2** — payment  
4. **Security Phase 2** sisa yang belum tertutup auth/quota user  

| Milestone | Hasil | Estimasi kalender |
|-----------|--------|-------------------|
| Hanya tahan abuse (tanpa monetisasi) | Rate limit + hardening OCR | **~3–5 hari** (hampir semua BE) |
| Login + kuota (belum bayar) | Tahan biaya Gemini per user | **~1.5–2 minggu** dari awal |
| First money (kuota + bayar) | Checkout QRIS + kredit | **~2.5–3.5 minggu** dari awal |

Jika FE & BE dikerjakan orang berbeda: FE mulai setelah kontrak API Monetization Phase 1 siap → kalender bisa lebih pendek ~20–30%.

---

## 3. Pecahan Frontend vs Backend

### 3.1 Security Phase 1 (~1–2 hari) — ✅ SELESAI

| Layer | Kerjaan | Estimasi | Status |
|-------|---------|----------|--------|
| **BE** | BodyLimit, timeout, rate limit IP, semaphore Gemini, CORS ketat, sanitize error, hide Swagger, reject upload oversized | 1–2 hari | **Done** (`release/v1.0.0`) |
| **FE** | Hampir tidak ada; opsional tampilkan pesan generik untuk HTTP 429 | ~0 | N/A |

→ Hampir **100% backend**. Selesai dikerjakan (termasuk BodyLimit 5 MiB, Groq failover, cleanup storage 14 hari).

---

### 3.2 Monetization Phase 0–1 (~1–1.5 minggu) — 🚧 IN PROGRESS

| Layer | Kerjaan | Estimasi | Status |
|-------|---------|----------|--------|
| **BE** | Postgres + migrasi + Wire; Google verify + JWT; `usage_monthly`; enforce kuota di `POST /api/v2`; `GET /me/quota` | 3–5 hari | Implemented (pending deploy/test) |
| **FE** | Modal/tombol Google login; simpan JWT; kirim `Authorization` di upload; tampil sisa kuota; handle 401 & 402 | 2–3 hari | Implemented (pending deploy/test) |

→ Bisa paralel: BE dulu endpoint auth/quota; FE sambung setelah API siap (atau mock sementara).

---

### 3.3 Monetization Phase 2 (~1 minggu)

| Layer | Kerjaan | Estimasi |
|-------|---------|----------|
| **BE** | Tabel `payments` + `credit_ledger`; checkout; webhook Midtrans/Xendit; grant kredit idempotent; poll status | 3–5 hari |
| **FE** | Modal beli `PACK_20`; tampil QR/redirect; poll/refresh kuota; lanjut upload setelah lunas | 2–3 hari |

→ FE tergantung checkout/webhook BE sudah jalan.

---

### 3.4 Security Phase 2 — sisa (~1–2 hari)

| Layer | Kerjaan | Estimasi |
|-------|---------|----------|
| **BE** | Magic-byte validation; storage TTL cleanup; API key / daily IP quota **hanya jika** masih dibutuhkan (banyak diganti login + kuota user) | 1–1.5 hari |
| **FE** | Nginx security headers; cek ukuran file sebelum upload | ~0.5 hari |

→ Mayoritas **backend**; FE kecil.

---

## 4. Timeline ringkas per milestone

| Milestone | BE | FE | Total kalender* |
|-----------|----|----|-----------------|
| Tahan abuse saja | 3–5 hari | ~0–0.5 hari | **3–5 hari** |
| Login + kuota (belum bayar) | +3–5 hari | +2–3 hari | **~1.5–2 minggu** dari awal |
| First money (sampai bayar) | +3–5 hari | +2–3 hari | **~2.5–3.5 minggu** dari awal |

\*Asumsi 1 orang full-stack, sequential sesuai §2.

---

## 5. Referensi fase di dokumen sumber

### security-sds.md

| Fase | Estimasi di SDS | Fokus | Status |
|------|-----------------|--------|--------|
| Phase 1 | 1–2 hari | Rate limit, size, timeout, concurrency, CORS, sanitize, Swagger | ✅ Selesai |
| Phase 2 | 2–4 hari | API key, daily quota IP, magic-byte, retention, FE headers | Partial (retention done) |
| Phase 3+ | Sedang–tinggi | Redis, CAPTCHA, WAF — di luar estimasi MVP ini | — |

### monetization-technical-design.md

| Fase | Fokus |
|------|--------|
| 0 | Postgres + migrasi |
| 1 | Google login, JWT, kuota, enforce OCR, UI kuota |
| 2 | Payment + kredit → first money |
| 3+ | Idempotency ketat, admin, monitoring, fitur belakangan — di luar estimasi MVP ini |

---

## Riwayat dokumen

| Tanggal | Perubahan |
|---------|-----------|
| 2026-09-12 | Tandai Security Phase 1 selesai; Monetization 0–1 in progress |
| 2026-09-12 | Draft awal estimasi + pecahan FE/BE + urutan praktis |
