# SIMS Backend

REST API untuk **NaviStock / SIMS** — Sistem Manajemen Inventaris Galangan Kapal.

---

## Tech Stack

| Komponen | Versi / Library |
|----------|-----------------|
| Bahasa | Go 1.25 |
| Web framework | Fiber v2 |
| Database | PostgreSQL (Supabase) |
| Driver | pgx/v5 |
| Migrasi | golang-migrate/v4 |
| Auth | JWT (golang-jwt/v5) + bcrypt |

---

## Prasyarat

Pastikan semua tool berikut sudah terinstall di komputer kamu:

| Tool | Versi Minimal | Cek dengan |
|------|--------------|------------|
| **Go** | 1.25.0 | `go version` |
| **Git** | — | `git --version` |

> **Catatan**: Kamu juga butuh akses ke project **Supabase** (gratis). Lihat bagian [Setup Supabase](#1-setup-supabase) di bawah.

---

## Cara Menjalankan (Dari Nol)

### 1. Clone Repository

```bash
git clone https://github.com/PROYEK3-ULBI/sims-backend.git
cd sims-backend
```

### 2. Setup Supabase

Jika belum punya project Supabase:

1. Buka [supabase.com](https://supabase.com) → **Start your project** (gratis)
2. Pilih region **Southeast Asia (Singapore)** untuk latency terbaik
3. **Catat database password** yang kamu buat — tidak bisa dilihat lagi nanti
4. Tunggu project selesai provisioning (~1 menit)
5. Buka **Project Settings → Database → Connection string**
6. Kamu butuh 2 connection string:
   - **Transaction pooler** (port `6543`) → untuk runtime aplikasi
   - **Session pooler** (port `5432`) → untuk migrasi database

> Jika kamu diberikan akses ke project Supabase yang sudah ada, minta kedua connection string ini ke pemilik project.

### 3. Konfigurasi Environment

```bash
# Salin template
cp .env.example .env
```

Buka file `.env` dan isi:

```env
# Transaction pooler — WAJIB diakhiri ?default_query_exec_mode=exec
DATABASE_URL=postgresql://postgres.XXXX:PASSWORD@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?default_query_exec_mode=exec

# Session pooler — untuk migrasi saja
DATABASE_MIGRATION_URL=postgresql://postgres.XXXX:PASSWORD@aws-0-ap-southeast-1.pooler.supabase.com:5432/postgres

# JWT secret — generate dengan: openssl rand -base64 32
# Atau buat string acak minimal 16 karakter
JWT_SECRET=ganti-dengan-string-acak-yang-panjang
```

> ⚠️ **Penting**:
> - Suffix `?default_query_exec_mode=exec` pada `DATABASE_URL` **wajib ada**. Tanpa ini, query akan error karena PgBouncer transaction mode tidak support prepared statements.
> - `JWT_SECRET` minimal 16 karakter, kalau kurang server akan gagal start.

### 4. Install Dependencies & Jalankan

```bash
# Download semua dependency Go
go mod tidy

# Jalankan server
go run ./cmd/api
```

Jika berhasil, kamu akan melihat log seperti ini:

```
[startup] migrations applied
[startup] database connected
[startup] server listening on :8080
```

Server berjalan di **http://localhost:8080**.

> 💡 **Koneksi pertama ke Supabase** bisa memakan waktu ~30 detik (cold start pooler). Ini normal.

### 5. Verifikasi

Cek server berjalan dengan membuka browser atau curl:

```bash
curl http://localhost:8080/healthz
```

Response: `{"status":"ok"}`

---

## CLI Tools Tambahan

Selain server utama, ada 2 CLI tool bawaan:

```bash
# Cek koneksi database tanpa menjalankan server
go run ./cmd/dbcheck

# Generate bcrypt hash untuk password
go run ./cmd/hashpw <password>
```

---

## Akun Demo

Setelah server pertama kali dijalankan, migration otomatis membuat 3 user demo:

| Role | Email | Password |
|------|-------|----------|
| Admin | `admin@shipyard.co.id` | `admin123` |
| Supervisor | `supervisor@shipyard.co.id` | `admin123` |
| Staff | `staff@shipyard.co.id` | `admin123` |

---

## Struktur Project

```
sims-backend/
├── cmd/
│   ├── api/          # Entry point server
│   ├── dbcheck/      # CLI cek koneksi database
│   └── hashpw/       # CLI generate bcrypt hash
├── internal/
│   ├── activitylog/  # Logging aktivitas user
│   ├── auth/         # Login, JWT, middleware, RBAC
│   ├── config/       # Loader konfigurasi dari .env
│   ├── database/     # Connection pool + migration runner
│   ├── materials/    # CRUD material/inventaris
│   ├── materialrequests/ # Workflow permintaan material
│   ├── notifications/    # Sistem notifikasi
│   ├── projects/     # Manajemen proyek/hull
│   ├── purchaseorders/   # Purchase Order
│   ├── reports/      # Laporan & analitik
│   ├── server/       # Bootstrap Fiber + error handler
│   ├── support/      # FAQ & bantuan
│   ├── tools/        # Manajemen alat kerja
│   ├── transactions/ # Transaksi masuk/keluar/scrap
│   ├── users/        # Manajemen pengguna
│   ├── vendors/      # Manajemen vendor
│   └── warehouselocations/ # Lokasi gudang
├── migrations/       # File SQL migrasi (000001..000020)
├── .env.example      # Template environment
├── Makefile          # Shortcut commands
├── go.mod
└── go.sum
```

Setiap modul di `internal/` mengikuti pola: **model.go → repository.go → service.go → handler.go**.

---

## Makefile Shortcuts

```bash
make run      # go run ./cmd/api
make build    # go build -o bin/api ./cmd/api
make tidy     # go mod tidy
make test     # go test ./...
make fmt      # gofmt -s -w .
```

---

## Strategi Koneksi Database

| Kegunaan | Pooler | Port | Env Variable |
|----------|--------|------|--------------|
| Runtime aplikasi | Transaction | 6543 | `DATABASE_URL` |
| Migrasi (DDL) | Session | 5432 | `DATABASE_MIGRATION_URL` |

Transaction pooler lebih efisien untuk production tapi tidak support advisory lock yang dibutuhkan oleh migration tool. Maka migrasi pakai session pooler.

---

## Troubleshooting

### Server gagal start — "JWT_SECRET too short"
`JWT_SECRET` di `.env` harus minimal 16 karakter.

### Error koneksi database
1. Jalankan `go run ./cmd/dbcheck` untuk verifikasi connection string
2. Pastikan suffix `?default_query_exec_mode=exec` ada di `DATABASE_URL`
3. Pastikan password Supabase benar (tidak ada karakter yang perlu di-escape?)
4. Supabase cold start bisa ~30 detik — tunggu dan coba lagi

### "migration failed" / advisory lock error
Pastikan `DATABASE_MIGRATION_URL` pakai **session pooler (port 5432)**, bukan transaction pooler.

### Port 8080 sudah dipakai
Ubah `APP_PORT` di `.env`:
```env
APP_PORT=9090
```

---

## API Format

Semua response error mengikuti format:

```json
{
  "error": "pesan error",
  "code": 400,
  "success": false
}
```

Semua endpoint (kecuali `/api/auth/login` dan `/healthz`) membutuhkan header:

```
Authorization: Bearer <jwt_token>
```

---

## Tim Pengembang

**PROYEK3-ULBI** — Universitas Logistik dan Bisnis Internasional
