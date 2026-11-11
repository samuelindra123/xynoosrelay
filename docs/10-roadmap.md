# Roadmap implementasi bertahap

[Indeks](README.md) | [Audit](01-audit-frontend.md) | [Standar](09-standar.md)

Dokumen ini adalah instruksi kerja untuk agent berikutnya, bukan izin untuk mengimplementasikan sekarang. Tiap fase menghasilkan PR terpisah dengan desain/kontrak dan tes; urutan dapat dipecah lebih kecil tetapi invariant dan security gate tidak boleh dilewati. Backend belum ada. Dokumentasikan seluruh asumsi dan keputusan sebelum membuka akses publik.

## Fase 1 — Foundation

- **Objective:** Go/Chi modular monolith dapat dibangun, dikonfigurasi dan diuji pada lingkungan aman.
- **Prerequisites:** tentukan owner proyek, Go version didukung runtime, region, domain, status penggunaan Hobby dan secret governance. Saat audit, Go tidak ditemukan pada PATH; toolchain perlu disediakan pada fase implementasi mendatang, tidak dipasang dalam audit ini.
- **Scope/deliverables:** struktur module boundary, satu adapter handler Vercel, router `/v1`, config tervalidasi, logging teredaksi, request ID, deadlines, error envelope, health endpoint, pgx/sqlc/goose setup, CI dan OpenAPI awal; database ephemeral untuk test.
- **Dependencies:** tidak ada fase lain; infrastruktur staging dan PoC runtime dari [07](07-realtime.md).
- **Acceptance:** build/test bersih, health dan routing PoC benar, invalid config fail-fast tanpa bocor secret, connection pool bounded dan migrasi serial, contract snapshot.
- **Risks:** Go runtime beta, rewrite/timeout, pool burst dan batas paket; rollback rencana dan compatibility database.
- **Done:** PR direview, CI reproducible, staging evidence jika ada, observability dan dokumentasi versi; tanpa mengklaim production readiness.

## Fase 2 — Authentication dan sessions

- **Objective:** autentikasi passwordless terverifikasi dan sesi dapat dicabut.
- **Prerequisites:** fase 1, domain same-site/CORS/CSRF PoC, provider email dan Google credentials staging, rate-limit store, security model [06](06-keamanan.md).
- **Scope/deliverables:** OTP challenge/resend/verify, email adapter, OIDC code+PKCE/state/nonce callback, identity linking policy, opaque sessions/device metadata, revoke/logout, OpenAPI dan audit events; tidak memakai kode demo.
- **Dependencies:** foundation DB, transaction coordinator, email/realtime-independent rate limiting.
- **Acceptance:** race verify hanya sekali, expiry/resend/attempt/rate limits server enforced, generic anti-enumeration, OIDC tampering ditolak, revocation langsung, CSRF/CORS/browser tested, session tidak bocor dalam log.
- **Risks:** brute force, email deliverability, account takeover/linking, cross-site browser restrictions dan recovery yang belum diputuskan.
- **Done:** security review + unit/integration/concurrency dan staging E2E termasuk multiple browser; belum memasang autentikasi produksi sebelum policy/consent ditetapkan.

## Fase 3 — Users dan profiles

- **Objective:** identitas stabil dan onboarding/profile persisten.
- **Prerequisites:** fase 2 dan keputusan username/privacy/recovery.
- **Scope/deliverables:** user/profile schema, username normalized unique, `/me`, onboarding, privacy-filtered user lookup, basic profile fields dan preference revision. Avatar/banner dapat tetap pending fase 7.
- **Dependencies:** sessions dan permissions untuk profile access.
- **Acceptance:** nama valid server, username race hanya satu sukses, onboarding idempotent, IDOR dan visibility tests, refresh menunjukkan data server, 409/422 terpresentasi di UI.
- **Risks:** enumeration, Unicode normalization dan perubahan username, retention data pribadi.
- **Done:** contract tests, migration rehearsal, accessibility/error states frontend integrasi masa depan dan audit dokumen.

## Fase 4 — Workspaces, channels dan permissions

- **Objective:** tenant isolation dan kolaborasi terotorisasi.
- **Prerequisites:** fase 3; capability matrix, owner transfer, channel visibility dan invitation policy disetujui.
- **Scope/deliverables:** atomic workspace/owner/default channel, membership, category/channel CRUD, effective ACL deny precedence, roles/capabilities, display title terpisah, membership profile overrides, invite create/preview/redeem/revoke, audit.
- **Dependencies:** users, sessions, transactional DB, rate limit.
- **Acceptance:** cross-tenant/private-channel read/write/search/subscribe ditolak; last owner tidak hilang saat race, custom title tidak mengubah izin; invite limit/revoke concurrency lolos; list/pagination tidak membocorkan resource.
- **Risks:** privilege escalation, confused deputy, owner lockout, invitation replay dan perubahan role saat request berjalan.
- **Done:** matrix positif/negatif otomatis dan peer security review, migration/rollback plan, OpenAPI lengkap dan UI error integration plan.

## Fase 5 — Messaging

- **Objective:** DM dan channel messaging durable, urut dan dapat dipulihkan tanpa realtime.
- **Prerequisites:** fase 4 untuk channel ACL; identitas, penghapusan, retention, read receipt privacy, message size dan moderation diputuskan.
- **Scope/deliverables:** DM canonical pair, participants, message sequence, keyset pagination, create/edit/delete tombstone, reply, pin, reactions, structured mentions, read cursor dan scoped search, idempotency/outbox.
- **Dependencies:** sessions, membership/permissions dan transaksi PostgreSQL.
- **Acceptance:** concurrent send memberi sequence unik per conversation, retry tidak menggandakan pesan, no cross-tenant IDOR, reply satu conversation, read cursor monoton, delete tidak membocorkan konten, pagination tanpa skip di concurrent insert.
- **Risks:** beban pencarian, urutan transaksi, stale ACL, idempotency replay dan pertumbuhan storage.
- **Done:** integration dan concurrency suite, contract contoh konkret, metrik dasar, functional REST meski realtime offline; tidak mengklaim E2EE.

## Fase 6 — Realtime

- **Objective:** percepat sinkronisasi tanpa menjadikan transport sumber kebenaran.
- **Prerequisites:** fase 5, PoC Go/WebSocket + pilihan provider/gate pada [07](07-realtime.md), target latency dan budget.
- **Scope/deliverables:** ticket authorize, subscription/revocation, outbox dispatcher, event envelope/version, dedupe, reconnect/backoff, snapshot+catch-up; presence/typing ephemeral jika disetujui.
- **Dependencies:** ledger replay aman atau fallback full resync; scheduler/queue yang sesuai batas plan.
- **Acceptance:** multi-instance, disconnect/reconnect, event duplicate/reorder, commit-vs-publish race dan permission revoke diuji; tidak ada missing durable state setelah catch-up; provider outage tetap memungkinkan REST.
- **Risks:** Go upgrade unsupported, timeout, fan-out authorization, lost event, biaya dan vendor lock-in.
- **Done:** PoC mencatat konfigurasi/versi/tanggal/evidence, load test, ADR pilihan transport, runbook backlog dan alert; bila PoC gagal gunakan fallback yang disetujui.

## Fase 7 — Media, stickers dan notifications

- **Objective:** upload dan ekspresi aman serta notifikasi yang menghormati preferensi.
- **Prerequisites:** fase 3, 5; storage/email policy, file size/type, scanning, licensing, retention, notification policy dan biaya disetujui.
- **Scope/deliverables:** private object store, scoped signed upload, verification/quarantine/finalize, owner ACL, orphan cleanup, profil/avatar/banner/attachment; sticker catalogue dengan hak penggunaan; notification inbox/preferences dan dedupe.
- **Dependencies:** media adapter, job runner/outbox, membership ACL, provider pendukung.
- **Acceptance:** upload unauthorized/oversized/spoofed tidak dapat dipakai, hanya ready assets direferensikan, signed URL tidak masuk log, ACL diperiksa pada read, notification tidak bocor setelah akses dicabut, quota/opt-out berfungsi.
- **Risks:** malware, object URL leakage, storage/egress cost, duplikasi notifikasi, image metadata dan vendor availability.
- **Done:** abuse/security test, lifecycle cleanup, licensed assets dan restore drill media, accessibility/UX states; push production belum diasumsikan.

## Fase 8 — Testing, security dan deployment hardening

- **Objective:** rilis aman dengan batas operasional yang terukur.
- **Prerequisites:** fase 1–7 sesuai lingkup rilis; kebijakan legal, ketersediaan dan billing disetujui.
- **Scope/deliverables:** E2E user journeys lintas device, load/soak, security review/pentest sesuai risiko, monitoring/alert, backup/restore, runbook incident dan rollback, upgrade/deprecation policy, SLO disepakati.
- **Dependencies:** staging representatif, data sintetis, CI/CD dan semua provider yang dipilih.
- **Acceptance:** auth/ACL/IDOR/OTP/race suite lolos, p95/p99 dan error rate memenuhi target yang disetujui, restore drill memenuhi RPO/RTO yang disetujui, no secrets/PII logs, failure injection dan provider outage teruji.
- **Risks:** asumsi kuota Hobby, kebocoran PII, backup tidak mencakup objects, cost meningkat dan target tidak realistis.
- **Done:** evidence, outstanding risks dan owner tercatat; security/product/ops menyetujui release. Jika staging/approval tidak tersedia, laporkan batas dan jangan sebut production-ready.

## Urutan tugas kecil untuk agent implementasi

1. Baca [audit](01-audit-frontend.md), [ADR](02-arsitektur.md), [API](05-api.md) dan [security](06-keamanan.md); cek source terkini karena dapat berubah.
2. Selesaikan open questions yang memblokir fase, tulis proposal konkret dan keputusan; jangan menebak secret/domain/SLA.
3. Buat satu vertical slice dengan kontrak, migrasi versi, test negatif dan observability; tidak sekaligus memindahkan semua state frontend.
4. Integrasi frontend per journey menggunakan adapter API, loading/error/retry dan feature flag; hapus hanya mock yang telah diganti secara teruji.
5. Lakukan contract/DB/concurrency/security tests, perbarui OpenAPI/ADR, tulis evidence dan risiko sebelum lanjut.
