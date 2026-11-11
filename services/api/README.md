# XYNOSS Relay — Backend Service (`services/api`)

Backend API resmi untuk **XYNOSS Relay**: *Messaging without the weight. Less app. More conversation.*

Dibangun menggunakan Go, Chi HTTP router, pgx/v5 connection pool, goose migrations, dan sqlc query integration. Backend ini dirancang sebagai modular monolith yang siap dideploy ke Vercel Functions maupun containerized environment (Docker/Kubernetes).

---

## 1. Struktur Direktori

```text
services/api/
├── api/
│   └── index.go             # Entrypoint Vercel Go Serverless Function (Handler)
├── bin/
│   └── server.exe           # Binary executable hasil kompilasi lokal
├── cmd/
│   └── server/
│       └── main.go          # Standalone HTTP server runner (local dev & container)
├── internal/
│   ├── config/              # Validasi konfigurasi environment & fail-fast
│   ├── database/            # Neon-compatible pgxpool connection manager & health probe
│   │   └── queries/         # sqlc queries source (queries.sql)
│   ├── health/              # Liveness, readiness, and status HTTP handlers
│   ├── logger/              # Structured JSON logging (slog) & redaksi data sensitif
│   ├── middleware/          # RequestID, CORS, SecurityHeaders, Timeout, Recoverer, Logger
│   ├── response/            # Standardized Success & Error envelopes (docs/05-api.md)
│   └── server/              # Chi router assembly & route mounting
├── migrations/
│   ├── 00001_initial_schema.sql  # Skema tabel database PostgreSQL lengkap (goose)
│   └── migrate.go           # Programmatic migration runner
├── openapi/
│   └── openapi.yaml         # Kontrak OpenAPI 3.1 REST API
├── .env.example             # Referensi variabel environment tanpa secret
├── go.mod                   # Deklarasi Go module dan dependencies
├── go.sum                   # Checksum cryptographic Go dependencies
├── sqlc.yaml                # Konfigurasi type-safe code generator sqlc
├── vercel.json              # Konfigurasi routing rewrite serverless Vercel
└── README.md                # Dokumentasi backend ini
```

---

## 2. Standar Respons & Error Envelope

Sesuai spesifikasi `docs/05-api.md`:

### Format Respons Sukses (HTTP 2xx)
```json
{
  "data": {
    "status": "ok"
  },
  "meta": {
    "next_cursor": "ey...",
    "has_more": false
  },
  "request_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6"
}
```

### Format Respons Error (HTTP 4xx / 5xx)
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "validation failed for one or more fields",
    "details": [
      { "field": "email", "code": "INVALID_FORMAT" }
    ],
    "request_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6"
  }
}
```

Daftar error code standar:
- `MALFORMED_REQUEST` (400)
- `UNAUTHENTICATED` (401)
- `FORBIDDEN` (403)
- `NOT_FOUND` (404)
- `CONFLICT` (409)
- `CURSOR_EXPIRED` (410)
- `REVISION_MISMATCH` (412)
- `PAYLOAD_TOO_LARGE` (413)
- `VALIDATION_ERROR` (422)
- `RATE_LIMITED` (429)
- `INTERNAL_ERROR` (500)
- `SERVICE_UNAVAILABLE` (503)

---

## 3. Prasyarat & Menjalankan Secara Lokal

### Prasyarat
- Go 1.23+ (Terpasang Go 1.27.3 di `C:\Users\Admin\go-sdk\go\bin`)
- PostgreSQL (Opsional untuk testing foundation; Neon PostgreSQL untuk staging/prod)

### Menjalankan Server Lokal
```powershell
cd services/api
go run ./cmd/server
```
Server akan mendengarkan pada `http://localhost:8080`.

### Verifikasi Endpoint
```powershell
# 1. Liveness Probe
curl.exe http://localhost:8080/health/live

# 2. Readiness Probe (Dependencies Check)
curl.exe http://localhost:8080/health/ready

# 3. Service Status
curl.exe http://localhost:8080/v1/health
```

---

## 4. Pengujian & Verifikasi Kualitas

Semua unit test berjalan secara terisolasi tanpa membutuhkan live database external:

```powershell
# Format code
go fmt ./...

# Static analysis
go vet ./...

# Unit test suite
go test -v ./...

# Build binary
go build -v -o bin/server.exe ./cmd/server
```

---

## 5. Deployment ke Vercel Functions

Struktur `api/index.go` dan `vercel.json` dirancang khusus untuk deployment serverless Vercel:
- `api/index.go` mengekspor fungsi `Handler(w http.ResponseWriter, r *http.Request)`.
- Router Chi, logger, dan connection pool diinisialisasi secara thread-safe pada cold start (`sync.Once`).
- `vercel.json` melakukan rewrite seluruh request `/(.*)` ke handler serverless `/api`.
