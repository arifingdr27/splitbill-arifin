# Database Migrator — Splitbill API

Migrator ini menerapkan perubahan schema PostgreSQL secara otomatis saat aplikasi start.

## Cara kerja

1. API connect ke Postgres (`DATABASE_URL`).
2. Pastikan tabel `schema_migrations` ada.
3. Baca semua file `internal/migrate/sql/*.up.sql` (ter-embed di binary).
4. File yang belum tercatat di `schema_migrations` dijalankan **berurutan** (nama file diurutkan).
5. Setiap file sukses → insert versi ke `schema_migrations` (dalam transaksi yang sama).

```
API start
   → postgres.NewPool
   → migrate.Runner.Up()
   → siap layani request
```

## Lokasi file

| Path | Fungsi |
|------|--------|
| `internal/migrate/sql/NNNNNN_nama.up.sql` | Migration maju (wajib) |
| `internal/migrate/sql/NNNNNN_nama.down.sql` | Rollback manual (opsional, belum dijalankan otomatis) |
| `internal/migrate/migrate.go` | Engine migrator |
| `migrations/` (root) | Mirror / catatan ops (opsional; sumber utama = `internal/migrate/sql`) |

Versi = nama file tanpa `.up.sql`, contoh: `000001_init`.

## Menambah migration baru

1. Buat file berikutnya (nomor naik):

```bash
# contoh
touch internal/migrate/sql/000002_add_foo.up.sql
touch internal/migrate/sql/000002_add_foo.down.sql
```

2. Isi `.up.sql` dengan SQL perubahan (hindari `CREATE` tanpa `IF NOT EXISTS` jika mungkin bentrok; untuk alter table biasanya langsung).

3. Isi `.down.sql` untuk rollback manual jika dibutuhkan.

4. Rebuild & deploy API — migrator jalan sendiri di startup.

**Aturan:**
- Jangan edit file `.up.sql` yang sudah pernah di-deploy ke production.
- Selalu buat file **baru** untuk perubahan berikutnya.
- Satu concern per migration (lebih mudah di-review).

## Cek status di database

```sql
SELECT * FROM schema_migrations ORDER BY version;
```

Atau:

```bash
docker exec -it shared_postgres psql -U shared -d splitbill \
  -c 'SELECT * FROM schema_migrations ORDER BY version;'
```

## Rollback manual

Otomatis **belum** support `migrate down`. Untuk rollback:

1. Jalankan isi file `.down.sql` secara manual di `psql`.
2. Hapus baris versi dari `schema_migrations`:

```sql
DELETE FROM schema_migrations WHERE version = '000002_add_foo';
```

Lakukan hati-hati di production.

## Shared-infra vs migrator app

| Komponen | Tugas |
|----------|--------|
| **shared-infra** | Menjalankan Postgres + volume + network; optional `CREATE DATABASE` |
| **Migrator Splitbill** | Mengubah **schema** di dalam DB `splitbill` (tabel users, quota, dll.) |

Infra tidak menggantikan migrator aplikasi.

## Troubleshooting

| Gejala | Kemungkinan | Perbaikan |
|--------|-------------|-----------|
| Gagal start: migration error | SQL salah / konflik | Perbaiki file **baru** atau perbaiki DB manual; jangan rewrite migration lama yang sudah applied |
| `migrations: nothing pending` | Semua sudah applied | Normal |
| Connect refused ke DB | shared-infra belum up / `DATABASE_URL` salah | `docker compose up -d` di shared-infra; cek host `shared_postgres` |
| DB `splitbill` tidak ada | Volume Postgres lama sebelum init script | `CREATE DATABASE splitbill;` manual |

## Contoh log sukses

```text
migration applied  version=000001_init
migrations: finished  count=1
```

atau:

```text
migrations: nothing pending
```
