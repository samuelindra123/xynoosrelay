# Infrastruktur, deployment dan operasi

[Indeks](README.md) | [Arsitektur](02-arsitektur.md) | [Realtime](07-realtime.md)

## Topologi dan environment

Frontend Next.js App Router project A pada Vercel Hobby; backend Go/Chi project B terpisah pada Vercel Functions; Neon PostgreSQL; layanan email dan object storage; realtime adapter opsional. Production, staging dan preview dipisah untuk database, secret, OAuth client/callback, cookie host, bucket serta provider namespace. Preview tidak boleh memakai data produksi. Project root backend, satu handler Go dan routing `/v1` dipastikan melalui PoC; jangan buat konfigurasi sekarang.

[Vercel Go runtime](https://vercel.com/docs/functions/runtimes/go) berstatus beta; satu handler di `/api`, runtime membaca versi dari go.mod project root. Tidak berarti routing Chi atau WebSocket Go sudah diverifikasi. [Hobby](https://vercel.com/docs/plans/hobby) personal/nonkomersial; rencana bisnis harus memeriksa kelayakan dan kemungkinan upgrade. [Limits umum](https://vercel.com/docs/limits) dan [WebSocket beta](https://vercel.com/changelog/websocket-support-is-now-in-public-beta) saling bertentangan; lihat [PoC](07-realtime.md). Limit/plan berubah: verifikasi dashboard dan halaman resmi sebelum memilih kapasitas.

## Environment variable yang direncanakan

Nama di bawah adalah usulan interface konfigurasi; **tidak membuat file env**. Secret hanya disediakan di backend/CI, tidak di client bundle maupun log.

| Nama | Scope dan tujuan | Wajib |
|---|---|---|
| APP_ENV, API_BASE_URL, WEB_ORIGINS | Backend: environment, URL/callback origin allowlist; validasi startup | Ya |
| DATABASE_URL | Backend: pooled TLS connection string Neon, least privilege | Ya |
| DATABASE_DIRECT_URL | Migrator CI saja: direct connection untuk goose bila sesuai; bukan client | Ya pada migrasi |
| SESSION_HASH_KEY, OTP_HMAC_KEYS | Backend secret berversi; hash verifier/codes dan rotasi key | Ya |
| GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URI | Backend OIDC client dan callback environment | Ya bila Google aktif |
| EMAIL_PROVIDER, EMAIL_API_KEY, EMAIL_FROM | Backend adapter dan alamat pengirim terverifikasi | Ya bila OTP aktif |
| STORAGE_ENDPOINT, STORAGE_BUCKET, STORAGE_ACCESS_KEY, STORAGE_SECRET_KEY | Backend signing terbatas ke bucket; terpisah per environment | Ya bila media aktif |
| REALTIME_PROVIDER, REALTIME_API_KEY, REALTIME_ENDPOINT | Backend provider adapter; ticket TTL minimum | Bila realtime aktif |
| RATE_LIMIT_STORE_URL, RATE_LIMIT_SECRET | Backend rate limiting terdistribusi; store tidak boleh in-memory saja | Ya untuk auth publik |
| OBSERVABILITY_ENDPOINT, OBSERVABILITY_TOKEN | Backend structured logs/errors dengan redaksi | Direkomendasikan |
| NEXT_PUBLIC_API_BASE_URL | Frontend: hanya URL publik API, tidak berisi secret | Ya saat integrasi |
| CRON_AUTH_SECRET | Dispatcher/cleanup scheduler: verifikasi request, rotasi | Bila scheduler HTTP digunakan |

Rahasia unik per environment; batasi akses dan rotasi. CORS dan cookies mengikuti [keamanan](06-keamanan.md). Backend memvalidasi required vars, format dan kombinasi sejak startup; jangan print DSN atau secret. Preview deploy API dengan callback preview memerlukan allowlist dinamis yang terkontrol, bukan wildcard.

## Database dan kapasitas

Gunakan pooled endpoint Neon untuk runtime dan direct endpoint terpisah untuk migrasi bila provider mensyaratkan. pgxpool per instance dengan `MaxConns` kecil dan `MinConns` rendah, acquire timeout, lifetime/jitter dan connection budget total = instances x max conns + migrator/operasional. Contoh awal 2 koneksi/instance hanya hipotesis, bukan sizing. [pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) menyediakan parameter tersebut.

Neon pooler transaction mode tidak menjamin session state antar transaksi; uji prepared statements/pgx/sqlc yang dipilih, penggunaan transaction-scoped settings dan incompatibilities, lalu dokumentasikan versi aktual. [Neon tentang pooling](https://neon.com/blog/inside-lubots-database-per-tenant-architecture). Uji autosuspend/wake, p95/p99 dan burst, tidak membuat pool baru per request, tidak menganggap koneksi instance tetap hidup.

## Provider pendukung — kandidat, bukan keputusan final

| Fungsi | Kandidat awal & alasan | Trade-off/validasi |
|---|---|---|
| Email OTP | Resend transactional API: integrasi HTTP sederhana; alternatif Postmark/SES | Domain verification, delivery/abuse/bounce, region, DPA, kuota, harga dan kualitas deliverability wajib diuji. Response generik tidak membuktikan delivery. |
| Media | Cloudflare R2 S3-compatible private bucket; alternatif storage setara | [Presigned PUT](https://developers.cloudflare.com/r2/api/s3/presigned-urls/) memberi satu operasi dengan bearer URL, bukan jaminan size/content. Uji CORS, biaya, scanning, lifecycle dan upload besar. |
| Rate limit | Store Redis managed dengan atomic operations; alternatif Postgres dengan tabel atomic untuk volume awal | Biaya/latency, fail-closed auth saat store down, anti-concurrency, retensi hash. Tidak gunakan memory per instance. |
| Realtime | Managed realtime atau polling REST; lihat [realtime](07-realtime.md) | Private subscriptions, revocation, quotas, cost dan region belum dipilih. |
| Logs/errors | Structured JSON + Vercel logs; error tracking redacted | Durasi retensi dan alert plan-dependent; jangan kirim PII, token, OTP atau isi pesan. |
| Background | Durable transactional outbox + authenticated scheduled dispatcher/managed queue | Function invocation bukan worker permanen; uji retry/lease, scheduler interval, budget dan recovery backlog. Cron Hobby perlu verifikasi frekuensi aktual; jika tidak cukup, external scheduler atau polling bukan janji realtime. |
| Cache | Tidak wajib di fase awal | Cache boleh untuk catalogue publik; jangan cache authorization tanpa invalidasi aman. Tambah setelah pengukuran. |

## Deploy, CI/CD dan observability

CI masa depan: format/static checks, unit, sqlc generation drift check, migration up/down atau forward-fix rehearsal pada DB ephemeral, integration tests, contract/OpenAPI compatibility, security scan, build dan staging PoC. Jangan menjalankan migrasi di setiap function startup. Satu migration job dengan lock dan least privilege; expand/contract kompatibel dua versi. Release API dan web independen; feature flags dengan default off. Production promotion setelah staging + smoke + rollback plan. Rollback code tidak otomatis rollback data; siapkan forward-fix.

Live health hanya proses; ready internal memeriksa dependency secara terbatas dan tidak mengungkap rahasia. Log JSON berisi request_id, actor pseudonymous, action, status, latency, deployment/region; redaksi PII. Metrics: request/error rate, p95/p99, cold start, DB acquire wait, pool saturation, OTP failure/delivery, rate limit, outbox age/retries, realtime reconnect, upload rejection. Alert berdasarkan baseline terukur, tidak mengklaim SLA; incident runbook mencakup isolate, revoke secrets/sessions, komunikasi, restore dan postmortem.

## Backup, recovery dan biaya

Verifikasi fitur dan durasi restore plan Neon aktual di dashboard; buat backup yang sesuai kebijakan, uji restore terisolasi dan ukur RPO/RTO; DB restore tidak otomatis mengembalikan object storage, email atau event provider. Inventaris asset dan recovery key terpisah; uji restore tiap periode yang disetujui. Neon branch bukan pengganti otomatis backup independen. Kebutuhan compliance/retention dan restore window final masih terbuka.

Vercel Hobby, Neon free tier dan provider pihak ketiga dapat berubah, habis kuota atau tidak mengizinkan skenario bisnis. Rp0 adalah target eksperimen, bukan jaminan produksi. Tetapkan owner billing, alert ambang, spending cap bila tersedia, model cost per active user/messages/storage/egress dan prosedur penghentian nonkritis. Jangan menaruh angka kuota sebagai janji permanen.
