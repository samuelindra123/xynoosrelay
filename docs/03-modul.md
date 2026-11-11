# Batas modul modular monolith

[Indeks](README.md) | [Data](04-data.md) | [API](05-api.md) | [Keamanan](06-keamanan.md)

Tabel hanya diubah owner modul. Kolom API merujuk keluarga route publik, bukan HTTP internal. Setiap modul punya service, repository interface, query sqlc, validasi dan tes; transaksi lintas modul dikoordinasikan application service. Semua operasi menggunakan context deadline, actor/session dan audit context.

| Modul | Tanggung jawab, data dan API | Dependensi, keamanan, gagal dan tes penting |
|---|---|---|
| Auth | OTP challenge dan external identity; `/auth/*`. Single-use, hasil generik. | Users, Sessions, email; HMAC, rate limit; uji race verify/resend, expiry, enumeration, provider gagal. |
| Users | Identitas internal dan username; `/me`, `/users/{id}`. Unik case-normalized. | Auth terpisah; uji race username, IDOR, deletion. |
| Profiles | Profil global dan override membership; `/me/profile`, `/workspaces/{id}/me/profile`. | Users, Media, Membership; uji sanitasi, fallback dan asset bukan milik user. |
| Sessions | Opaque sessions, expiry dan revocation; `/me/sessions`. | Auth, Devices; cookie/CSRF; uji revoke paralel, current session dan expiry. |
| Devices | Label dan metadata perkiraan dari session; `/me/sessions`. | Tidak gunakan fingerprint sebagai autentikasi; uji UA palsu/unknown. |
| Workspaces | Workspace dan lifecycle; `/workspaces`; create + owner + default channel atomik. | Membership, Channels, audit; uji rollback, last owner, cross-tenant. |
| Channels | Channel dan kategori; `/workspaces/{id}/channels`, `/categories`. | Permissions/Conversations; uji slug, private-channel leak di list/search. |
| Membership | Anggota aktif/left; `/workspaces/{id}/members`. | Users, Roles; uji double join, self escalation, last owner, revoked access. |
| Roles | Permission role dan display title berbeda; `/workspaces/{id}/roles`. | Permissions; uji escalation, custom role, transfer owner. |
| Permissions | Registry capability + override; internal authorize(actor, action, resource), default deny. | Membership, Roles, Channels; uji matrix lintas tenant dan revoke. |
| Invitations | Token hash, TTL, cap, revoke; `/workspaces/{id}/invitations`, `/invitations/*`. | Membership/Permissions; uji double redeem, expiry, limit, preview privacy; rahasia tidak masuk log. |
| Conversations | DM pair atau channel binding, participants dan read cursor; `/conversations/*`. | Users, Membership, Channels; uji pembuatan paralel, akses dicabut dan cursor monoton. |
| Messages | Konten, reply, edit, tombstone, pin, sequence; `/conversations/{id}/messages`. | Conversations, Permissions, outbox, Media; uji retry, edit conflict, reply lintas ruang. |
| Reactions | Satu user+emoji per message; `/messages/{id}/reactions`. | Messages/ACL; uji duplicate retry, concurrency, revoke. |
| Stickers | Catalogue published dan pack; `/stickers`. | Media, licensing; uji asset tidak tersedia dan unauthorized. |
| Notifications | Inbox dan delivery state; `/me/notifications`. | Messages/Settings; uji dedupe, opt-out, spam dan akses dicabut. |
| Realtime | Envelope, subscription authorization, adapter dan outbox dispatcher. | Uji duplicate, reconnect, reorder, gap, revoke subscription dan provider outage. |
| Media | Upload intent, ownership, verify/scan/finalize; `/media/*`. | Storage adapter; uji MIME spoof, size, expiry, orphan dan cross-user. |
| Settings | Preference per user/workspace; `/me/settings`. | Users/Notifications; whitelist fields; uji PATCH parsial, default dan lost update. |

## Permission baseline — usulan

Owner: semua termasuk transfer ownership; Admin: kelola channel, invite dan role yang tidak melebihi dirinya; Moderator: moderasi sesuai policy; Member: baca/tulis channel yang diizinkan dan kelola pesan sendiri. Capability contoh: `workspace.manage`, `members.manage`, `roles.manage`, `channel.manage`, `channel.read`, `channel.write`, `message.moderate`, `invite.manage`. Explicit channel deny mengalahkan allow; tanpa hak efektif = deny. Owner bypass masih perlu keputusan produk; invariant last owner tetap wajib. Role kustom memberi capability; display title tidak pernah memberi hak. Semua contoh perlu product/security review.
