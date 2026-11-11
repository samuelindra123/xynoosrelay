# Kontrak REST v1 — proposal

[Indeks](README.md) | [Keamanan](06-keamanan.md) | [Model data](04-data.md)

Belum ada API/OpenAPI file. Tabel berikut adalah kontrak kerja untuk diubah menjadi OpenAPI setelah keputusan terbuka diselesaikan. Semua path memiliki prefix `/v1`. Nama field snake_case. `id` adalah UUID string, timestamps RFC3339 UTC; tidak menerima actor, permission atau timestamp sebagai kebenaran dari client.

## Konvensi lintas endpoint

- HTTPS; `Content-Type: application/json`, `Accept: application/json`; upload object memakai signed URL sesuai instruksi media. Kecuali ditandai Publik, setiap endpoint perlu sesi aktif; mutation berbasis cookie juga CSRF + Origin. Semua ID resource diauthorize server. Notasi **A** = sesi, **P** = publik; **M** = active workspace member + akses channel bila relevan; **C(cap)** = capability efektif, **O** = pemilik resource.
- Response sukses objek `{ data, meta?: { next_cursor, has_more }, request_id }`; delete tanpa body 204 bila dinyatakan. List default 30 max 100; `cursor` opaque scope-bound, tidak memberi hak. Endpoint non-list tidak memiliki pagination. Waktu/order server authoritative.
- Error `{ error: { code, message, details?: [{ field, code }], request_id } }`. Umum: 400 malformed, 401 `UNAUTHENTICATED`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 409 `CONFLICT`, 410 `CURSOR_EXPIRED`, 412 `REVISION_MISMATCH`, 413 `PAYLOAD_TOO_LARGE`, 422 `VALIDATION_ERROR`, 429 `RATE_LIMITED` + `Retry-After`, 503 `SERVICE_UNAVAILABLE`. Resource privat dapat mengembalikan 404; jangan membedakan akun pada auth publik. Tidak ada stack trace/secrets.
- Batas per endpoint didefinisikan kategori **auth**, **read**, **write**, **invite**, **upload**, **realtime** di [keamanan](06-keamanan.md); semua rate limit server, kuota final belum diputuskan. Abuse berdasarkan IP+actor+resource sesuai risiko. 429 berlaku untuk semua; tabel menyebut tambahan khusus.
- `Idempotency-Key` wajib pada create yang berpotensi menghasilkan side effect eksternal/duplikat dan ditandai **I**; scoped actor+operation, fingerprint body, respons sama pada retry; mismatch 409. PUT/DELETE yang bersifat set/remove alami idempotent; conditional PATCH memakai revision/If-Match jika disebut. Idempotency tidak menggantikan permission check saat replay.
- Nama fungsi, batas, dan default yang belum disetujui merupakan proposal. Saat integrasi buat contoh JSON dan definisi penuh dalam OpenAPI, termasuk nullable/required/enum dan semua kode error; tabel ini belum file OpenAPI.

## Authentication dan account

| Method + path | Tujuan dan auth/permission | Request -> data response | Error, rate, idempotency |
|---|---|---|---|
| POST `/auth/otp/challenges` | P; minta kode atau resend | `email` -> 202 `challenge_id, retry_after_seconds` generik; challenge opaque, waktu respons tidak membocorkan akun | auth/IP+email; 422 format, 429; I, invalidasi challenge sebelumnya atomik |
| POST `/auth/otp/verifications` | P + CSRF pre-auth; verify | `challenge_id, code` -> `user{id,display_name}, onboarding_required, csrf_token` + Set-Cookie sesi | auth; invalid/expired/consumed respons generik, 429; consume atomik, replay tidak membuat sesi kedua |
| GET `/auth/google/start` | P; mulai OAuth | opsional return_path allowlisted -> redirect ke provider; pre-auth state cookie | auth; 400 path, 429, 503 provider |
| GET `/auth/google/callback` | P; callback OAuth | `code,state` -> server redirect ke web allowlisted + Set-Cookie | invalid state/provider 400, 429; state one-use; tidak log code |
| GET `/auth/session` | A; bootstrap | tidak ada -> `user, onboarding_required, csrf_token` | read; 401 expired/revoked |
| POST `/auth/logout` | A; revoke current | tidak ada -> 204 + clear cookie | write; idempotent pada sesi aktif/revoked |
| GET `/me` | A | tidak ada -> `id, username, onboarding_required, profile` | read |
| POST `/me/onboarding/complete` | A; penyelesaian onboarding | `display_name, username?` -> `user, onboarding_required:false` | write; 409 username; I |
| GET `/users/{user_id}` | A; hanya profil yang boleh ditemukan | tidak ada -> `id, username, display_name, avatar, status` | read; 404 privacy/nonexistent |
| GET `/users` | A; penemuan pengguna sesuai kebijakan | `q,cursor,limit` -> list profil ringkas | read ketat/anti-enumeration; 422 q |
| GET `/me/profile` | A/O | tidak ada -> `display_name, username, bio, avatar_media_id, banner_media_id, availability, custom_status, revision` | read |
| PATCH `/me/profile` | A/O | subset field di atas + `If-Match` revision -> profil diperbarui | write; 409 username, 412 revision, 422 invalid/asset not ready |
| GET `/me/sessions` | A/O | `cursor,limit` -> `id, label, platform_hint, last_seen_at, is_current, expires_at` | read; hint bukan device attestation |
| DELETE `/me/sessions/{session_id}` | A/O; revoke | tidak ada -> 204; sesi current juga hapus cookie | write; idempotent; 404 bukan milik |
| GET/PATCH `/me/settings` | A/O | GET -> `appearance, notifications, privacy, accessibility, revision`; PATCH subset + If-Match -> settings | read/write; 412 revision, 422; daftar fields final perlu keputusan |

## Workspaces, akses dan undangan

| Method + path | Tujuan dan auth/permission | Request -> data response | Error, rate, idempotency |
|---|---|---|---|
| GET `/workspaces` | A; hanya membership aktif | `cursor,limit` -> workspace ringkas + effective capabilities | read |
| POST `/workspaces` | A | `name, description?, slug?` -> 201 `workspace, membership, default_channel` | write; 409 slug, 422; I; satu transaksi |
| GET `/workspaces/{workspace_id}` | A/M | tidak ada -> `id,name,slug,description,owner_id,capabilities` | read; 404 nonmember |
| PATCH `/workspaces/{workspace_id}` | A/M/C(workspace.manage) | `name?,description?,revision` -> workspace | write; 409/412 revision |
| DELETE `/workspaces/{workspace_id}` | A/Owner + re-auth sesuai policy | `If-Match` -> 204 soft-delete | write; 409 legal hold, I kebijakan konfirmasi final |
| GET/POST `/workspaces/{workspace_id}/categories` | A/M read; POST C(channel.manage) | GET list `cursor,limit`; POST `name,position?` -> category | read/write; POST I, 409 duplicate |
| PATCH/DELETE `/workspaces/{workspace_id}/categories/{category_id}` | A/M/C(channel.manage) | PATCH `name?,position?,revision` -> category; DELETE -> 204 | write; 409 jika tidak kosong sesuai policy, 412 revision |
| GET/POST `/workspaces/{workspace_id}/channels` | A/M; POST C(channel.manage) | GET `cursor,limit` -> hanya channel visible; POST `name,slug,category_id?,visibility` -> channel + conversation ID | read/write; 409 slug, 422 category; POST I |
| GET/PATCH/DELETE `/workspaces/{workspace_id}/channels/{channel_id}` | GET A/M/channel.read; PATCH/DELETE C(channel.manage) | GET -> channel; PATCH `name?,slug?,category_id?,position?,visibility?,revision` -> channel; DELETE -> 204 | read/write; 404 private, 409/412, delete policy TBD |
| GET/PUT `/workspaces/{workspace_id}/channels/{channel_id}/acl` | A/M/C(channel.manage), hierarchy | GET -> ACL entries; PUT `entries[{subject_type,subject_id,capability,effect}]` -> revision | read/write; 409 escalation/last-admin, 422 cross-tenant; PUT idempotent |
| GET `/workspaces/{workspace_id}/members` | A/M; privacy-filter | `cursor,limit,q?` -> `user, permission_roles, display_title, nickname, avatar` | read; 404 private |
| PATCH/DELETE `/workspaces/{workspace_id}/members/{user_id}` | A/M/C(members.manage), hierarchy | PATCH `role_ids?, display_title?, nickname?` -> membership; DELETE -> 204 | write; 409 last owner/escalation, 404; DELETE idempotent |
| GET/PATCH `/workspaces/{workspace_id}/me/profile` | A/M; sendiri | GET -> `nickname,display_title,avatar_media_id`; PATCH subset -> override | read/write; title policy dan revision perlu diputuskan |
| GET/POST `/workspaces/{workspace_id}/roles` | A/M; POST C(roles.manage) | GET list; POST `name,capabilities[]` -> role | read/write; 409 duplicate/escalation; POST I |
| PATCH/DELETE `/workspaces/{workspace_id}/roles/{role_id}` | A/M/C(roles.manage), hierarchy | PATCH `name?,capabilities?,revision` -> role; DELETE -> 204 | write; 409 system role/in-use/escalation, 412 revision |
| GET/POST `/workspaces/{workspace_id}/invitations` | A/M/C(invite.manage) | GET `cursor,limit` -> metadata tanpa secret; POST `expires_at,max_uses,role_id?` -> 201 `id,token_once,expires_at` | invite; 422/403 role escalation; POST I dan replay handling secret perlu desain aman |
| DELETE `/workspaces/{workspace_id}/invitations/{invite_id}` | A/M/C(invite.manage) | tidak ada -> 204 revoke | invite; idempotent |
| POST `/invitations/preview` | P; token dalam body | `token` -> informasi minimum `workspace_name, icon?, available` | invite/IP; 404 invalid generic, 429; token jangan di URL/log |
| POST `/invitations/redeem` | A | `token` -> `workspace_id,membership` | invite; 404 invalid/expired/revoked generic, 409 limit/already-member sesuai policy; I |

**Token sekali tampil dan idempotency:** jangan menyimpan token plaintext dalam tabel idempotency atau log. Rancangan awal: simpan respons token yang dienkripsi dengan kunci terpisah dan TTL singkat untuk replay terotorisasi, atau respons retry menyatakan token tidak dapat ditampilkan ulang. Pilihan dan UX harus diputuskan sebelum implementasi; tidak mengklaim keduanya sudah aman tanpa review.

## Percakapan dan konten

| Method + path | Tujuan dan auth/permission | Request -> data response | Error, rate, idempotency |
|---|---|---|---|
| GET/POST `/conversations` | A; GET participant/visible; POST DM | GET `cursor,limit,q?` -> list + unread; POST `participant_user_id` -> conversation | read/write; 404 privacy; POST I + unique pair |
| GET `/conversations/{conversation_id}` | A/DM participant atau M/channel.read | tidak ada -> `id,kind,participants_or_channel,last_sequence,unread_count` | read; 404 unauthorized |
| GET/POST `/conversations/{conversation_id}/messages` | A + read/write sesuai jenis | GET `before_sequence?,limit,q?` -> messages desc, next_cursor; POST `client_message_id,kind,text?,sticker_id?,media_ids?,reply_to_id?,mentions[]?` -> 201 `message{id,sequence,author,created_at,revision}` | read/write; 409 duplicate key/reply mismatch, 422 invalid, POST I; ACL sebelum pencarian dan limit |
| PATCH/DELETE `/conversations/{conversation_id}/messages/{message_id}` | A + author atau C(message.moderate) menurut policy | PATCH `text,revision` -> message; DELETE -> 204 tombstone | write; 403/404, 409 revision, 422; retry DELETE idempotent |
| PUT/DELETE `/messages/{message_id}/reactions/{emoji}` | A + conversation access | tanpa body -> PUT reaction agregat, DELETE 204 | write; 422 unsupported; idempotent per actor/emoji |
| GET/PUT/DELETE `/conversations/{conversation_id}/pins/{message_id}` | A + read; mutations pin permission | GET -> pin; PUT -> pin; DELETE -> 204 | read/write; idempotent PUT/DELETE |
| GET `/conversations/{conversation_id}/pins` | A + read | `cursor,limit` -> pinned messages yang masih terlihat | read; cursor |
| POST `/conversations/{conversation_id}/read` | A + read | `last_read_sequence` -> `last_read_sequence,unread_count` | write; 422 future sequence; monotonic/idempotent, client baru clear setelah ack |
| GET `/stickers/packs` | A | `cursor,limit` -> packs published + sticker IDs/media | read; visibility sesuai licensing |
| GET `/stickers/{sticker_id}` | A | tidak ada -> `id,pack_id,media_id,description` | read; 404 unpublished |

## Media, notifications dan realtime

| Method + path | Tujuan dan auth/permission | Request -> data response | Error, rate, idempotency |
|---|---|---|---|
| POST `/media/uploads` | A; intent | `purpose,content_type,size_bytes,checksum?` -> `media_id,upload_url,method,required_headers,expires_at` | upload; 413/422, quota 429; I; URL secret, expiry singkat |
| POST `/media/{media_id}/finalize` | A/O | `checksum?` -> `status` pending/quarantined/ready/rejected | upload; 409 invalid state; I; verifikasi object sebelum ready |
| GET `/media/{media_id}` | A + resource ACL | tidak ada -> metadata + URL baca pendek bila authorized | read; 404 unauthorized/unready; jangan cache URL secara publik |
| DELETE `/media/{media_id}` | A/O + not referenced sesuai policy | tidak ada -> 204 | upload; 409 referenced, idempotent |
| GET `/me/notifications` | A/O | `cursor,limit` -> notifications yang masih dapat diakses | read; cursor |
| POST `/me/notifications/{notification_id}/read` | A/O | tidak ada -> `read_at` | write; idempotent |
| POST `/realtime/tickets` | A + CSRF | `subscriptions[{kind,id}]` -> `ticket,expires_at,endpoint` setelah ACL | realtime; 403/422, TTL singkat, no token logging; provider/transport belum final |
| GET `/events` | A; scope ACL | `cursor,limit` -> durable events yang masih authorized + next_cursor | read; 410 cursor expired -> full resync; tidak memberikan presence ephemeral |
| GET `/health/live` | P, data minimal | tidak ada -> `status` | read/rate; tidak mengungkap secret |
| GET `/health/ready` | internal/monitor terotorisasi | tidak ada -> `status,dependencies` | read; 503 dependency; hindari DB query berlebihan |

## Kebijakan respons yang perlu diputuskan

Penggunaan 404 vs 403 untuk resource privat; ukuran maksimum body dan teks; algoritme search; edit window; approval owner bypass; nama HTTP header CSRF; aturan 410 cursor expired; endpoint pembaruan permission dan ownership transfer granular; route device rename. Seluruh penambahan harus backward-compatible dalam v1 atau melalui v2 dengan deprecation period terukur.
