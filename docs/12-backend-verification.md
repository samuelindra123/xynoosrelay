# XYNOSS RELAY — LAPORAN VERIFIKASI & PENGIRIMAN BACKEND (FASE 2)

> **Proyek:** XYNOSS Relay Backend (`services/api`)  
> **Tanggal:** 8 November 2026  
> **Status Akhir:** **VERIFIED PASS (100% Tests Passing & Live Neon PostgreSQL Integration)**  
> **Insinyur:** Principal Golang Backend Engineer, Security Engineer & DevOps Architect  

---

## 1. Ringkasan Eksekutif

Sesuai dengan instruksi pengerjaan backend XYNOSS Relay, implementasi **Autentikasi Penuh**, **Media Object Storage Terisolasi**, **CORS & Keamanan Header**, **Komunikasi Internal Aman (Service-to-Service)**, dan **API Pengguna & Profil** telah selesai dibangun secara nyata, bukan sekadar dokumentasi atau stub.

Seluruh kode backend telah melewati verifikasi ketat melalui:
1. **Unit Testing** pada seluruh modul internal (`auth`, `config`, `database`, `health`, `middleware`, `response`, `session`, `storage`, `users`).
2. **Integration Testing End-to-End** secara langsung terhadap **Neon PostgreSQL Live Database** (`ep-billowing-unit-76511277`).
3. **Uji Keamanan & Hardening** mencakup pencegahan IDOR, proteksi CSRF, anti-account enumeration, proteksi race condition (atomic row locks), anti-decompression bomb, sniffing magic bytes, dan constant-time cryptographic comparison.
4. **Kompilasi Sukses** binary server produksi: `services/api/bin/server.exe` (ukuran 26.9 MB, Go 1.26.0).

---

## 2. Evaluasi Teknis: Better Auth vs Native Go Authentication

Sesuai permintaan evaluasi arsitektur, dilakukan perbandingan mendalam antara **Option A (Better Auth sebagai dedicated TypeScript microservice)** dan **Option B (Native Go Authentication Engine)**.

### Perbandingan Arsitektur

| Aspek Evaluasi | Option A: Better Auth (TypeScript Microservice) | Option B: Native Go Authentication (Dipilih & Diimplementasikan) |
|---|---|---|
| **Kompatibilitas Runtime** | Mengharuskan penambahan runtime Node.js/Bun terpisah (`services/auth`) di samping Go backend. | 100% terpadu dalam satu runtime Go modular monolith yang ringan dan cepat. |
| **Domain Cookies pada Vercel** | **Bermasalah di Vercel**. Subdomain `*.vercel.app` terdaftar dalam *Public Suffix List (PSL)*. Cookie tidak dapat dibagikan lintas subdomain Vercel (`auth.vercel.app` ke `api.vercel.app`) tanpa custom domain. | Bebas masalah domain cookie. Endpoint `/v1/auth/*` berada di origin API yang sama. |
| **Latensi & Network Hop** | Setiap request autentikasi membutuhkan network hop antar serverless function Vercel (+150-300ms overhead). | Zero network hop internal. Sesi divalidasi langsung di memori/database pgx connection pool. |
| **Konsistensi Skema Database** | Memerlukan tabel khusus Better Auth yang berpotensi redundan atau berkonflik dengan 19 tabel Neon yang sudah termigrasi rapi. | 100% selaras dengan skema Neon yang sudah ada (`users`, `identities`, `sessions`, `devices`, `otp_challenges`). |
| **Cold-Start Performance** | Runtime Node.js/TypeScript memiliki cold start 300-800ms di Vercel. | Runtime Go memiliki cold start instan (<50ms). |
| **Kontrol Keamanan Rendah-Level** | Terbatas pada konfigurasi library; sulit menyatukan atomic `SELECT ... FOR UPDATE` dengan business logic pengguna. | Kontrol penuh atas kriptografi CSPRNG, constant-time comparison, HMAC-SHA256, dan transaksi database atomik. |

### Keputusan Teknis
**Dipilih Option B (Native Go Authentication Engine)**. Pendekatan ini menghilangkan kompleksitas mengelola dua backend (TypeScript dan Go), menghemat biaya hosting serverless, mencegah kegagalan sesi cookie di lingkungan `*.vercel.app`, serta memberikan performa tinggi dan keamanan kriptografi yang sepenuhnya dapat diaudit.

---

## 3. Fitur yang Telah Diimplementasikan

### 3.1. Mesin Autentikasi (`internal/auth`)
- **Email OTP 6-Digit (Passwordless):**
  - Pembuatan kode OTP menggunakan CSPRNG dengan rentang seragam `100000 - 999999`.
  - Kode disimpan dalam bentuk HMAC-SHA256 salted hash; tidak pernah ada plain OTP di database.
  - Rate limiting berjenjang: Cooldown kirim ulang 60 detik per email, maksimal 5 percobaan verifikasi per challenge, masa berlaku OTP 10 menit.
  - Proteksi Race Condition: Menggunakan `SELECT ... FOR UPDATE` saat verifikasi untuk memastikan kode tidak dapat digunakan ganda (*one-time use atomic invalidation*).
  - Anti-Account Enumeration: Endpoint permintaan OTP selalu mengembalikan HTTP `202 Accepted` baik email sudah terdaftar maupun baru.
- **Google OAuth 2.0 / OIDC:**
  - Pembuatan URL otorisasi dengan PKCE (RFC 7636, code verifier & S256 challenge).
  - State token bertanda tangan kriptografis HMAC dengan TTL 15 menit untuk mencegah serangan CSRF pada callback OAuth.
  - Callback handler untuk pertukaran authorization code, sinkronisasi identitas Google, dan pembuatan sesi instan.
- **Session Management:**
  - Token sesi 256-bit CSPRNG berformat `<session_id>.<verifier_secret>`.
  - Verifikasi token menggunakan SHA-256 verifier hash dengan perbandingan waktu-konstan (`subtle.ConstantTimeCompare`).
  - Sesi mendukung masa tenggang idle 7 hari dan absolute expiry 30 hari, dengan perpanjangan otomatis saat aktif (*sliding expiration*).
  - Cookie HTTP-only: `__Host-xynoss_session` pada produksi dan `xynoss_session` pada development.
  - Fitur Logout yang mencabut sesi aktif dari database secara permanen.

### 3.2. Lapisan Penyimpanan Media (`internal/storage` & `internal/media`)
- **Abstraksi Storage Multi-Provider (`StorageProvider`):**
  - **Local Disk Provider:** Menyimpan file secara aman di disk lokal dengan signed direct-upload token, proteksi path traversal, dan pembersihan file otomatis.
  - **S3 / R2 / Neon Object Storage Provider:** Mendukung AWS S3, Cloudflare R2, atau Neon Object Storage menggunakan AWS Signature Version 4.
  - **Mock Storage Provider:** Thread-safe provider untuk unit testing tanpa dependensi I/O disk.
- **Validasi Keamanan Media Tingkat Tinggi (`validator.go`):**
  - **MIME Allowlist:** Hanya menerima `image/jpeg`, `image/png`, dan `image/webp`.
  - **Magic Bytes Sniffing:** Membaca signature byte awal (`\xFF\xD8\xFF`, `\x89PNG\r\n\x1a\n`, `RIFF...WEBP`) untuk menggagalkan file berbahaya yang dinamai ekstensi gambar palsu.
  - **Penolakan Skrip & SVG:** Mendeteksi dan memblokir tag berbahaya (`<svg`, `<?xml`, `<html`, `<script`) untuk mencegah serangan Cross-Site Scripting (XSS).
  - **Anti-Decompression Bomb:** Membaca header dimensi gambar secara stream (`image.DecodeConfig`) dan menolak gambar dengan resolusi di atas **4096 x 4096 piksel**.
  - **Batas Ukuran:** Avatar maksimal **5 MB**; Banner maksimal **10 MB**.
- **Alur Unggah 4-Langkah (Intent & Finalize):**
  1. `POST /v1/media/uploads` membuat tiket intent dan signed URL.
  2. Pengguna mengunggah payload biner langsung ke storage provider via `PUT`.
  3. `POST /v1/media/{media_id}/finalize` memicu inspeksi keamanan oleh backend dan mengubah status menjadi `ready`.
  4. Pengguna menautkan media ke avatar atau banner profil. Media lama otomatis dihapus dari storage.
  5. **Proteksi IDOR:** Hanya pemilik media sah yang dapat memfinalisasi atau menautkan aset ke profilnya.

### 3.3. Middleware Keamanan (`internal/middleware` & `internal/authctx`)
- **CORS Allowlist Ketat:**
  - Hanya menerima origin terdaftar (`http://localhost:3000`, `http://127.0.0.1:3000`, dan domain produksi).
  - Mengizinkan credentials (`Access-Control-Allow-Credentials: true`).
  - Mengizinkan custom header: `X-CSRF-Token`, `X-Internal-API-Key`, `Idempotency-Key`, `If-Match`.
- **Proteksi CSRF Wajib (`RequireCSRF`):**
  - Memvalidasi header `X-CSRF-Token` pada seluruh mutasi (`POST`, `PUT`, `PATCH`, `DELETE`) jika pengguna terautentikasi melalui cookie sesi.
  - Menggunakan HMAC session-bound CSRF token yang tidak dapat dipalsukan oleh domain eksternal.
- **Internal API Key Middleware (`RequireInternalAPIKey`):**
  - Digunakan untuk komunikasi aman antar-layanan (service-to-service).
  - Memvalidasi header `X-Internal-API-Key` menggunakan `subtle.ConstantTimeCompare` guna mencegah serangan *timing attack*.
  - Mendukung rotasi kunci tanpa downtime melalui konfigurasi multiple-key terpisah koma.

### 3.4. API Pengguna & Profil (`internal/users`)
- `GET /v1/me`: Menampilkan profil lengkap, status onboarding, daftar identitas terhubung, dan preferensi akun.
- `POST /v1/me/onboarding/complete`: Menyelesaikan tahap registrasi onboarding dengan menetapkan nama tampilan dan username.
- `PATCH /v1/me/profile`: Memperbarui profil secara fleksibel dengan validasi:
  - Regex Username: `^[a-zA-Z0-9_]{3,32}$`.
  - Panjang Bio: maksimal 280 karakter.
  - Optimistic Concurrency Control: Mendukung header `If-Match: "<revision>"` untuk mencegah tumpang tindih data.
- `POST /v1/users/me/avatar` & `DELETE /v1/users/me/avatar`: Pasang dan hapus avatar profil.
- `POST /v1/users/me/banner` & `DELETE /v1/users/me/banner`: Pasang dan hapus banner profil.
- `GET /v1/me/sessions` & `DELETE /v1/me/sessions/{id}`: Audit dan cabut akses sesi perangkat lain.

---

## 4. Matriks Bukti Pengujian (Test Verification Results)

Semua rangkaian pengujian dijalankan dengan toolchain resmi Go 1.26.0 di lingkungan Windows x64.

### 4.1. Hasil Unit & Integration Test Suite (`go test ./...`)

| Paket Pengujian | Durasi | Status | Cakupan Pengujian |
|---|---|---|---|
| `internal/auth` | 10.48s | **PASS** | Validasi OTP, proteksi brute force rate limit, atomic lock, PKCE & OAuth state, hash matching. |
| `internal/config` | 3.08s | **PASS** | Parsing env vars, validasi URL database, deteksi mode environment, default fallback. |
| `internal/database` | 6.42s | **PASS** | Connection pool pgx, Neon scale-to-zero cold start resilience, migration readiness. |
| `internal/health` | <0.1s | **PASS** | Health check status (`/v1/health`), liveness probe, database ping. |
| `internal/middleware` | 5.31s | **PASS** | CORS preflight, CSRF enforcement, Internal API Key constant-time verification & key rotation. |
| `internal/response` | <0.1s | **PASS** | Format envelope JSON, error code builder, RFC-compliant error structure. |
| `internal/session` | <0.1s | **PASS** | Pembuatan token CSPRNG, hash verification, parse user-agent hint, sliding expiry. |
| `internal/storage` | <0.1s | **PASS** | Sanitasi path traversal, deteksi magic bytes, batas ukuran file, pencegahan zip/decompression bomb. |
| `internal/users` | 2.97s | **PASS** | Validasi regex username, optimistic revision control, sanitasi profil. |
| `internal/server` (Live Neon DB E2E) | 23.16s | **PASS** | **End-to-End Live Integration**: OTP Challenge -> Verify -> Sesi Cookie -> CSRF -> Onboarding -> Profile Update -> Session List -> Logout -> 401 Rejection. |

### 4.2. Bukti Log Eksekusi Uji Integrasi Nyata

```text
=== RUN   TestAPI_EndToEndAuthAndProfileFlow
--- PASS: TestAPI_EndToEndAuthAndProfileFlow (11.77s)
=== RUN   TestAPI_CSRFProtectionEnforcement
--- PASS: TestAPI_CSRFProtectionEnforcement (0.00s)
=== RUN   TestAPI_InternalAPIKeyProtection
--- PASS: TestAPI_InternalAPIKeyProtection (0.00s)
PASS
ok      github.com/xynoss/relay/services/api/internal/server    23.157s
```

### 4.3. Verifikasi Kompilasi Binary

```powershell
go build -o bin/server.exe ./cmd/server
```
- **File Output:** `services/api/bin/server.exe`
- **Ukuran File:** 26,904,064 bytes (~26.9 MB)
- **Status:** Berhasil tanpa warning, siap dideploy atau dijalankan secara lokal.

---

## 5. Ringkasan Endpoint & Kontrak OpenAPI

File kontrak OpenAPI 3.1 di `services/api/openapi/openapi.yaml` telah diperbarui mencakup seluruh endpoint baru:

| Method | Endpoint | Auth | CSRF | Keterangan |
|---|---|---|---|---|
| `POST` | `/v1/auth/otp/challenges` | Publik | Tidak | Minta kode OTP 6-digit (Anti-enumeration) |
| `POST` | `/v1/auth/otp/verifications` | Publik | Tidak | Verifikasi kode OTP, set cookie sesi & csrf_token |
| `GET` | `/v1/auth/google/start` | Publik | Tidak | Memulai flow Google OAuth PKCE |
| `GET` | `/v1/auth/google/callback` | Publik | Tidak | Callback OAuth, set cookie sesi & redirect |
| `GET` | `/v1/auth/session` | Sesi | Tidak | Bootstrap sesi & ambil user info + csrf_token baru |
| `POST` | `/v1/auth/logout` | Sesi | **Ya** | Revoke sesi aktif & hapus cookie di browser |
| `GET` | `/v1/me` | Sesi | Tidak | Ambil data lengkap pengguna saat ini |
| `POST` | `/v1/me/onboarding/complete` | Sesi | **Ya** | Selesaikan onboarding (username + display name) |
| `GET` | `/v1/me/profile` | Sesi | Tidak | Ambil profil detail pengguna |
| `PATCH` | `/v1/me/profile` | Sesi | **Ya** | Perbarui field profil (dengan dukungan `If-Match`) |
| `POST` | `/v1/users/me/avatar` | Sesi | **Ya** | Tautkan media siap pakai sebagai avatar profil |
| `DELETE` | `/v1/users/me/avatar` | Sesi | **Ya** | Hapus avatar profil pengguna |
| `POST` | `/v1/users/me/banner` | Sesi | **Ya** | Tautkan media siap pakai sebagai banner profil |
| `DELETE` | `/v1/users/me/banner` | Sesi | **Ya** | Hapus banner profil pengguna |
| `GET` | `/v1/me/sessions` | Sesi | Tidak | Daftar seluruh sesi dan perangkat aktif |
| `DELETE` | `/v1/me/sessions/{session_id}` | Sesi | **Ya** | Cabut akses sesi perangkat lain |
| `POST` | `/v1/media/uploads` | Sesi | **Ya** | Minta intent upload media (Avatar/Banner) |
| `POST` | `/v1/media/{media_id}/finalize` | Sesi | **Ya** | Finalisasi dan validasi file sniffing |
| `GET` | `/v1/internal/ping` | Internal | Tidak | Health check terproteksi `X-Internal-API-Key` |

---

## 6. Dokumen Pendukung & Panduan untuk Codex AI

Untuk memfasilitasi tim frontend (Codex AI), telah dibuat panduan integrasi komprehensif:
- **`docs/11-frontend-integration.md`**:
  - Alur lengkap autentikasi OTP & Google OAuth.
  - Format request, response, dan struktur penanganan error.
  - Alur upload direct-to-storage 4 langkah untuk Avatar dan Banner.
  - Contoh implementasi helper client TypeScript (`apiClient`) lengkap dengan manajemen cookie dan header `X-CSRF-Token`.

---

## 7. Kesimpulan & Langkah Selanjutnya

Pekerjaan fase backend ini telah selesai dengan **tingkat keberhasilan 100%**. Tidak ada mocking data palsu atau compiler palsu; seluruh integrasi telah terhubung ke Neon PostgreSQL dan lulus pengujian kriptografi, konkurensi, dan keamanan.

**Rekomendasi Langkah Berikutnya:**
1. **Frontend Integration:** Codex AI dapat mengimplementasikan form login OTP, integrasi sesi, dan manajemen profil pada `apps/client` sesuai panduan di `docs/11-frontend-integration.md`.
2. **Fase 3 Backend:** Memulai implementasi modul Workspace, Kategori, Channel, dan Realtime Messaging sesuai roadmap `docs/10-roadmap.md`.
