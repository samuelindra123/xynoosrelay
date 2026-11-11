# Realtime: transport, jaminan dan validasi

[Indeks](README.md) | [API](05-api.md) | [Data](04-data.md) | [Operasi](08-infrastruktur.md)

## Fakta, konflik sumber dan hal yang belum terbukti

- [Go runtime Vercel](https://vercel.com/docs/functions/runtimes/go) menyatakan Go runtime beta dan satu HTTP handler. Ini tidak membuktikan WebSocket upgrade, `http.Hijacker`, Go library tertentu, atau kompatibilitas Chi.
- [Changelog Vercel 22 Juni 2026](https://vercel.com/changelog/websocket-support-is-now-in-public-beta) mengumumkan WebSocket public beta pada Fluid compute dan menunjukkan Node.js; [panduan Vercel](https://vercel.com/kb/guide/real-time-chat-websockets) menjelaskan batas masa hidup function dan reconnect. [Halaman limits](https://vercel.com/docs/limits) masih menyatakan WebSocket server tidak didukung. Sumber resmi bertentangan; jangan menyimpulkan dukungan Go dari contoh Node.
- **Belum diuji:** Go upgrade pada runtime akun/plan/region target, library Chi/Go, connection pinning/duration aktual, limit concurrency, biaya, provider latency dan deployment. Belum ada hasil PoC pada proyek ini.

## Keputusan awal

DB dan REST adalah sumber kebenaran; realtime adalah akselerator yang boleh gagal. Mulai dengan polling REST ber-interval adaptif sebagai baseline dan external managed provider untuk bidirectional/push ketika dibutuhkan. Buat adapter internal untuk authorize/publish/revoke; ticket singkat dengan channel scopes terkecil. Pilih provider berdasarkan private channels, auth webhook, disconnect/revocation, region, volume, quota, billing dan DPA; jangan mengunci produk pada SDK sebelum uji. Contoh kandidat Pusher, Ably, Supabase Realtime atau layanan websocket terpisah; kompatibilitas dan harga harus diverifikasi saat memilih, bukan asumsi.

Jika PoC Go lulus dan biaya/operasi mendukung, native Go WebSocket dapat dipertimbangkan sebagai adapter. Function instance tidak boleh menjadi sumber kebenaran untuk room, fan-out atau sesi; koneksi bisa berakhir pada batas durasi. SSE + POST adalah alternatif server-push, tetap bounded oleh durasi; polling aman tetapi lebih lambat dan menambah request. Penyedia eksternal menambah vendor, secret dan biaya. Server long-running adalah perubahan deployment yang memerlukan ADR dan persetujuan arsitektur.

## Model event

Durable event envelope usulan: `event_id` UUID, `schema_version`, `type`, `conversation_id`, `message_sequence` bila berlaku, `occurred_at`, `payload` minimum. Event: `message.created/updated/deleted`, `reaction.changed`, `pin.changed`, `read.updated`, `membership.changed`, `channel.changed`, `notification.created`. Event tidak membawa cookie/token/OTP atau konten sensitif yang tidak perlu. Authorize recipient setiap publish/subscription; identitas workspace dalam payload bukan bukti izin.

Outbox ditulis pada transaksi domain yang sama; dispatcher claim pakai lease, retry backoff+jitter, dead-letter/alert dan idempotent publication. Publish sukses lalu crash sebelum ack bisa mengirim duplikat; client dedupe `event_id`, apply revision monoton dan tetap reconcile REST. Per-conversation sequence hanya mengurutkan **pesan**, bukan seluruh jenis event; stream_event_sequence terpisah mengurutkan event domain dalam stream, sedangkan edit/delete/reaction tetap butuh resource revision atau fetch resource. Race subscribe vs REST diselesaikan dengan subscribe dulu, buffer event, fetch snapshot/head, apply dan dedupe berdasarkan revision, lalu catch-up lagi.

### Recovery yang tidak melewatkan commit

Jangan memakai timestamp atau auto-increment ID yang dialokasikan sebelum commit sebagai global event cursor: transaksi dapat commit terbalik. `GET /events` di [API](05-api.md) hanya boleh dijanjikan setelah ledger replay per-user/per-stream dengan urutan commit aman, retention, filter ACL saat baca, dan pagination dibuktikan. Salah satu usulan adalah sequence per-user yang dialokasikan di bawah row lock hingga transaksi commit; fan-out besar memerlukan desain lock ordering dan pengujian beban. Jangan mengekspos cursor global lintas tenant.

Sebelum ledger itu siap, recovery wajib re-fetch daftar conversation dan latest messages menggunakan sequence per conversation, plus re-fetch read/pins/metadata. Koneksi dapat melewatkan edit/delete untuk pesan lama: snapshot revision atau ledger replay yang benar diperlukan sebelum menjanjikan recovery penuh. Cursor expired -> full resync; dedupe dan hak akses diperiksa lagi. Ini adalah gate, bukan solusi yang sudah diimplementasikan.

## Tabel kebutuhan

| Kebutuhan | Model yang dituju | Kegagalan / tes |
|---|---|---|
| DM dan channel | Server authorize tiap conversation; publish setelah commit. | Cross-tenant/private channel tidak bocor; race leave vs event; reconnect dan revocation. |
| Read receipt | Update cursor monoton via REST, event hanya jika policy mengizinkan. | Request lama tidak memundurkan cursor; server membatasi sequence head. |
| Presence | Ephemeral per session/device, TTL + heartbeat; tidak dijadikan bukti online pasti. | Multi-device disconnect, TTL, suspend, provider outage; privacy opt-out. |
| Typing | Event ephemeral dengan TTL pendek dan rate limit; jangan disimpan sebagai message. | Stop hilang/duplikat, throttle dan akses dicabut. |
| Delivery ack | 201 berarti persisted, bukan dibaca/dikirim ke perangkat. Delivery/read terpisah bila produk menyetujui. | Retry POST dengan idempotency; ack event bisa terlambat. |
| Ordering | Sequence ditetapkan DB per conversation di bawah lock; event dapat reorder. | Parallel write, pagination, duplicates dan snapshot/replay. |
| Reconnect | Exponential backoff+jitter, re-auth, subscribe ulang, catch-up; token singkat. | Offline lama, expiry cursor, revocation, switch workspace dan multi-tab. |
| Horizontal fan-out | External broker/provider + per-user authorization; no in-memory global state. | Instance freeze/restart, burst, outbox backlog dan partial failure. |

## PoC gate sebelum implementasi production

1. Pada project Vercel backend **staging** yang dibuat pada fase mendatang, deploy satu Go handler + Chi dan cek HTTP routes, CORS/cookie, cold start dan timeout.
2. Uji upgrade, ping/pong, reconnect, frame limit, disconnect, max duration, simultan clients dan actual Go library dengan logging metadata aman; gunakan dua instance/client dan uji fan-out.
3. Uji revoke membership/session selama koneksi, ticket expiration, replay/outbox duplicate, network loss dan snapshot reconciliation.
4. Ukur p95 latency, penggunaan connection DB, invocation/duration, quota dan biaya; verifikasi term Hobby. Catat versi runtime, region, tanggal, konfigurasi, hasil dan batas yang diamati.
5. Jika salah satu aspek unsupported atau biaya tidak cocok, pilih external provider atau polling dengan ADR. Tidak mengganti Go/Chi/Neon hanya karena asumsi.

## Open questions

Provider final, region/data residency, volume/presence concurrency, event retention, permissible delivery latency, read receipt privacy, apakah typing di DM/channel, dan definisi SLA. Jangan menjanjikan exactly-once atau selalu online.
