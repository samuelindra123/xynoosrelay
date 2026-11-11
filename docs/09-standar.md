# Standar engineering dan tata kelola

[Indeks](README.md) | [Modul](03-modul.md) | [Roadmap](10-roadmap.md)

## Struktur Go yang diusulkan

Satu entrypoint Vercel tipis; package internal untuk bootstrap/config, HTTP router/middleware, auth, users/profiles, sessions/devices, workspaces/membership/roles/permissions/channels/invitations, conversations/messages/reactions, media/stickers, notifications/settings, realtime/outbox dan observability. Tiap module memisahkan domain/application, repository interface dan adapter persistence/transport. Transaction coordinator berada pada application layer, tanpa circular dependency. Peta folder final diputuskan saat implementasi sesuai runtime Vercel; jangan membuat file sekarang.

## Standar

| Area | Aturan |
|---|---|
| Go/dependencies | Versi Go yang didukung Go Vercel dikunci saat PoC; module versions dan checksum dikunci, upgrade terencana, govulncheck/security scanning. Jangan menginstal Go pada fase dokumentasi. |
| HTTP | Resource plural, lower-case, `/v1`, id opaque, JSON snake_case, PATCH parsial, no verbs kecuali action seperti redeem/read. OpenAPI kontrak terversi; breaking changes melalui versi baru. |
| Error | Typed domain errors diterjemahkan sekali ke envelope [API](05-api.md); tidak membocorkan driver, SQL, secret atau detail existence akun. Jangan menelan context cancellation. |
| Context | Propagasi request context, cancel dan deadline ke db/provider; batas ukuran body dan timeout. Detached worker hanya dari durable job, tidak mengandalkan goroutine setelah function selesai. |
| Logging | Structured log request_id, event, duration, sanitized actor/resource; redaksi OTP, cookies, URLs bertoken, isi pesan dan PII. Audit terpisah dari debug. |
| Config | Satu loader tervalidasi, environment eksplisit, no secret default; dependensi di-inject, secret rotation terencana. |
| Data | sqlc query parameterized; pgx pool reuse, transaksi singkat, lock order jelas; goose versioned immutable; no schema changes saat startup. |
| Validation | Server validasi format, canonicalization, ukuran, enum dan referensi; DB constraints adalah lapisan kedua. Permission selalu server-side dan tenant-scoped. |
| Tests | Unit invariant; integration Neon/Postgres ephemeral + migration; API contract; authorization matrix lintas tenant; concurrency idempotency/OTP/invite/order; reconnect/outbox; failure injection dan security regression. No test yang mengandalkan vendor produksi. |
| Security review | Threat model tiap perubahan auth/media/ACL; pemeriksaan OWASP, rate limit, secrets scan, dependency scan, audit event, failure modes, recovery dan privacy. |
| Code review | Reviewer memeriksa invariant, batas transaksi, cancellation, logs, rollback dan test evidence. Tidak ada jaminan keamanan dari UI demo. |
| Dokumentasi | ADR untuk perubahan keputusan; version OpenAPI dan perubahan migrasi; tautkan docs/roadmap ke PR. Tandai asumsi dan tanggal verifikasi provider. |

## Definition of done lintas fase

Acceptance criteria [roadmap](10-roadmap.md) selesai; unit/integration/contract dan negative auth tests hijau; observability teredaksi; perubahan backward compatible atau ada rencana migrasi; staging smoke dan rollback/recovery dibuktikan pada environment yang tersedia; docs/OpenAPI diperbarui; security review pada perubahan berisiko. Jika infrastruktur belum tersedia, laporkan eksplisit; jangan menyatakan siap produksi hanya karena build.
