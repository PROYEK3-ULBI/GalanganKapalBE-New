# SIMS Backend

REST API untuk **NaviStock / SIMS** — Sistem Manajemen Inventaris Galangan Kapal.

## Tech Stack

- **Go 1.25** + **Fiber v2** (web framework)
- **PostgreSQL** via Supabase (database)
- **pgx/v5** (driver)
- **golang-migrate** (schema migrations)
- **JWT** + **bcrypt** (authentication)

## Project Structure

```
sims-backend/
├── cmd/
│   ├── api/main.go        # Server entry point
│   └── hashpw/main.go     # CLI helper: generate bcrypt hash
├── internal/
│   ├── auth/              # Auth module (login, JWT, middleware, users)
│   ├── config/            # Env-based configuration loader
│   ├── database/          # Connection pool + migrations runner
│   └── server/            # Fiber app bootstrap + error handler
├── migrations/            # Versioned SQL migrations
├── .env.example           # Template - copy to .env
├── go.mod
├── go.sum
└── Makefile
```

## Setup

### 1. Install Go 1.25+

Pastikan `go version` minimal 1.25.0.

### 2. Buat Project Supabase

1. Login ke [supabase.com](https://supabase.com), buat project baru
2. Pilih region **Southeast Asia (Singapore)**
3. Simpan database password yang dibuat
4. Setelah project ready, buka **Project Settings → Database → Connection string**
5. Copy **Transaction pooler** URI dan **Session pooler** URI

### 3. Konfigurasi Environment

```bash
cp .env.example .env
```

Edit `.env` dan isi:

- `DATABASE_URL` — Transaction pooler (port 6543), tambah `?default_query_exec_mode=exec` di akhir
- `DATABASE_MIGRATION_URL` — Session pooler (port 5432)
- `JWT_SECRET` — generate dengan `openssl rand -base64 32`

### 4. Jalankan

```bash
go mod tidy
go run ./cmd/api
```

Server akan otomatis:
1. Apply migrations (buat tabel `users` + seed 3 demo user)
2. Connect ke database
3. Listen di `http://localhost:8080`

## API Endpoints

### Auth

| Method | Path | Auth | Body |
|---|---|---|---|
| `POST` | `/api/auth/login` | Public | `{ email, password }` |
| `GET` | `/api/auth/me` | Bearer JWT | — |
| `GET` | `/healthz` | Public | — |

## Demo Credentials

Setelah migrations berjalan:

| Role | Email | Password |
|---|---|---|
| Admin | `admin@shipyard.co.id` | `admin123` |
| Supervisor | `supervisor@shipyard.co.id` | `admin123` |
| Staff | `staff@shipyard.co.id` | `admin123` |

## Connection Pool Strategy

- **Runtime** pakai **Transaction Pooler (port 6543)** — efisien untuk Cloud Run
- **Migrations** pakai **Session Pooler (port 5432)** — butuh advisory lock

## Deployment ke Cloud Run

(Coming soon — Dockerfile + deployment guide)
