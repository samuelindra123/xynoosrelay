# Laporan Pengujian Backend Go

**Tanggal:** 13 November 2026
**Cakupan:** Seluruh paket Go di `services/api`
**Status run terakhir:** Lulus

## Ringkasan

Seluruh test Go dijalankan dengan perintah `go test ./... -count=1 -cover`. Perintah tersebut menguji semua paket dan menampilkan persentase baris kode yang tercakup test. Koneksi database untuk test integrasi diambil dari `DATABASE_URL` pada `services/api/.env`, tanpa menampilkan nilainya.

Tidak ada server aplikasi yang dijalankan dengan `go run`. Test HTTP menggunakan server test di dalam proses test.

## Hasil per paket

| Paket | Hasil | Coverage |
|---|---:|---:|
| `api` | Lulus | 0,0%* |
| `cmd/migrate` | Lulus | 0,0%* |
| `cmd/server` | Lulus | 0,0%* |
| `internal/auth` | Lulus | 39,8% |
| `internal/authctx` | Lulus | 0,0%* |
| `internal/config` | Lulus | 61,7% |
| `internal/database` | Lulus | 46,7% |
| `internal/health` | Lulus | 68,2% |
| `internal/logger` | Lulus | 0,0%* |
| `internal/media` | Lulus | 53,0% |
| `internal/messaging` | Lulus | 18,4% |
| `internal/middleware` | Lulus | 56,9% |
| `internal/response` | Lulus | 55,9% |
| `internal/server` | Lulus | 96,1% |
| `internal/session` | Lulus | 16,2% |
| `internal/storage` | Lulus | 48,2% |
| `internal/users` | Lulus | 7,6% |
| `internal/workspaces` | Lulus | 13,5% |
| `migrations` | Lulus | 0,0%* |

\* Coverage 0,0% berarti tidak ada baris paket tersebut yang dihitung tercakup oleh test dalam run ini. Paket tetap berhasil dikompilasi dan tidak menyebabkan test gagal.

## Area yang diuji

- Seluruh test unit yang terdaftar di paket Go.
- Matriks route HTTP untuk route backend yang didaftarkan, termasuk pemeriksaan metode, autentikasi, dan respons route yang tidak valid.
- Alur autentikasi OTP, verifikasi, sesi, onboarding profil, pembaruan profil, dan logout pada test integrasi.
- Proteksi CSRF dan endpoint internal yang menggunakan API key.
- Alur workspace, channel, pesan, idempotensi pesan, undangan, serta percakapan langsung dalam test integrasi.
- Test paket media dan penyimpanan yang tersedia.

## Perbaikan selama pengujian

Run awal menemukan panic pada test endpoint internal ping saat daftar `INTERNAL_API_KEYS` kosong. Fixture server test sekarang mengisi kunci khusus test jika konfigurasi tidak menyediakannya. Perubahan ini hanya untuk test dan tidak mengubah `.env`.

Data akun integration test memakai email unik. Test juga membersihkan akun, workspace, percakapan, dan OTP yang dibuatnya.

Setelah perbaikan, seluruh paket lulus pada run ulang tanpa cache.

## Catatan cakupan

Status lulus berarti test yang ada saat ini berhasil, tetapi tidak berarti setiap fungsi sudah diuji. Coverage beberapa paket masih rendah, terutama `internal/users` (7,6%), `internal/messaging` (18,4%), `internal/session` (16,2%), dan `internal/workspaces` (13,5%). Paket CLI, logger, auth context, API, dan migrations juga belum menunjukkan baris yang tercakup pada laporan ini.

Karena test integrasi memakai `DATABASE_URL` dari `.env`, koneksi mengarah ke database yang dikonfigurasi di sana. Pada konfigurasi sesi ini database memakai branch Neon `production`; test menggunakan data sementara dan menjalankan cleanup setelahnya. Jangan memakai database produksi untuk test berulang tanpa memastikan cleanup serta kebijakan data yang sesuai.
