# Migrator di aplikasi (salinan)

**Sumber kebenaran** sekarang ada di repo **shared-infra**:

```text
shared-infra/migrator/splitbill/*.sql
```

Baca: `shared-infra/migrator/README.md`

Folder `sql/` di sini adalah salinan untuk auto-migrate saat API start (embed binary).  
Jika menambah migration: **tulis dulu di shared-infra/migrator/splitbill/**, lalu sync file ke sini.

```bash
# contoh dari mesin lokal
cp ../shared-infra/migrator/splitbill/*.sql internal/migrate/sql/
```

Jalankan migrator shared-infra (disarankan sebelum/ bersamaan deploy):

```bash
cd ../shared-infra
./migrator/migrate.sh splitbill up
```
