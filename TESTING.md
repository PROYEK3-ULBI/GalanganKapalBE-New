# Cara Cek Backend SIMS

## 1. Cek Koneksi Database (Tanpa Start Server)

Gunakan utility `dbcheck` untuk verify connection string di `.env`:

```powershell
go run ./cmd/dbcheck
```

Output yang benar:
```
[DATABASE_MIGRATION_URL (migration, 5432)]
  ✓ connected
  ✓ PostgreSQL 17.6 ...

[DATABASE_URL (runtime, port 6543)]
  ✓ connected
  ✓ PostgreSQL 17.6 ...
```

Kalau salah satu gagal, cek password di `.env`.

## 2. Start Server

```powershell
go run ./cmd/api
```

Tunggu sampai muncul:
```
[startup] migrations applied
[startup] database connected
[startup] listening on :8080 (env=development)
```

> Note: First request setelah server idle bisa lambat (cold start pooler). Request berikutnya cepat.

## 3. Cara Test Endpoint

### Opsi A — Pakai REST Client di VS Code (Paling Mudah)

1. Install extension **"REST Client"** by Huachao Mao
2. Buka file `requests.http` di project ini
3. Klik link **"Send Request"** di atas setiap blok request
4. Response muncul di panel sebelah kanan
5. Token hasil login otomatis ter-capture, jadi bisa langsung test endpoint protected

### Opsi B — PowerShell + curl

Buka PowerShell **baru** (server tetap running di terminal lain):

```powershell
# Login dan simpan token
$resp = curl.exe -s -X POST http://localhost:8080/api/auth/login `
  -H "Content-Type: application/json" `
  -d '{\"email\":\"admin@shipyard.co.id\",\"password\":\"admin123\"}'
$token = ($resp | ConvertFrom-Json).token
Write-Host "Token: $token"

# Test list materials
curl.exe -s "http://localhost:8080/api/materials" -H "Authorization: Bearer $token"

# Test filter HAZMAT
curl.exe -s "http://localhost:8080/api/materials?hazmat=true" -H "Authorization: Bearer $token"

# Test search
curl.exe -s "http://localhost:8080/api/materials?search=AH36" -H "Authorization: Bearer $token"
```

### Opsi C — Postman / Thunder Client

1. Login: `POST http://localhost:8080/api/auth/login`
   - Body (JSON): `{ "email": "admin@shipyard.co.id", "password": "admin123" }`
   - Copy `token` dari response
2. Endpoint protected: tambahkan header
   - `Authorization: Bearer <paste-token-disini>`

## 4. Cek Data Langsung di Supabase

1. Buka **Supabase Dashboard** kamu
2. Klik **Table Editor** di sidebar
3. Pilih tabel `users` atau `materials`
4. Bisa edit data langsung dari UI Supabase
5. Untuk SQL custom: pakai **SQL Editor**

Contoh query yang berguna:
```sql
-- Lihat semua users
SELECT id, email, name, role, status, last_login_at FROM users;

-- Lihat material dengan stok di bawah minimum
SELECT sku, name, stock, min_stock FROM materials WHERE stock <= min_stock;

-- Hitung material per kategori
SELECT category, COUNT(*) FROM materials GROUP BY category;
```

## 5. Debug Common Issues

| Error | Penyebab | Solusi |
|---|---|---|
| `tenant/user not found` | Connection string masih placeholder `YOUR_REF` | Edit `.env`, ganti dengan project ref asli |
| `password authentication failed` | Password salah | Reset password di Supabase, update `.env` |
| `ping database: timeout` | Network slow / cold start pooler | Sudah ditangani — timeout 30s. Jika masih, cek koneksi internet |
| `bind: Only one usage of each socket address` | Port 8080 sudah dipakai server lain | Tutup server lama dulu (`Ctrl+C`), atau ganti `APP_PORT` di `.env` |
| `Cannot GET /api/auth/login` di browser | Login butuh `POST`, bukan `GET` | Pakai REST Client / curl / Postman |
| `401 missing authorization header` | Lupa kirim header Authorization | Tambahkan `Authorization: Bearer <token>` |
| `403 insufficient permissions` | Role tidak punya akses (mis. supervisor coba create material) | Login sebagai admin |

## 6. Endpoint Reference

| Method | Path | Auth | Role |
|---|---|---|---|
| GET | `/healthz` | — | — |
| POST | `/api/auth/login` | — | — |
| GET | `/api/auth/me` | Bearer | any |
| GET | `/api/materials` | Bearer | any |
| GET | `/api/materials/categories` | Bearer | any |
| GET | `/api/materials/:id` | Bearer | any |
| POST | `/api/materials` | Bearer | admin |
| PUT | `/api/materials/:id` | Bearer | admin |
| DELETE | `/api/materials/:id` | Bearer | admin |

Query params di `GET /api/materials`:
- `?search=plat`
- `?category=Steel Plates`
- `?hazmat=true`
- `?lowStock=true`

Bisa digabung: `?category=Steel Plates&lowStock=true`
