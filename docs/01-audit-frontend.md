# Audit frontend dan pemetaan backend

[Indeks](README.md) | [API](05-api.md) | [Model data](04-data.md)

## Metode dan batas

Inspeksi statis source di `apps/client/src/app`, `package.json`, `next.config.ts` dan README, termasuk perubahan lokal yang belum di-commit. Tidak ada pengujian browser atau backend dalam audit ini. **Implemented UI** = tampilan ada; **Interactive mock** = perilaku dalam React memory; **Partially implemented** = sebagian alur; **Planned only** = target tanpa bukti implementasi; **Not found** = tidak ditemukan dalam source.

README root tertinggal: masih menyebut tombol akun sebagai placeholder, sedangkan `auth-flow.tsx` dan `/app` sudah ada. Tidak ada `/open-source`; halaman tersebut sebelumnya dihapus.

## Struktur, route, dan state

| Area / lokasi | Status dan fakta |
|---|---|
| `apps/client/package.json`, `next.config.ts` | Next 16, React 19, TypeScript, Tailwind 4; cacheComponents/partialPrefetching aktif. Tidak ditemukan klien API atau state library. |
| `page.tsx`, `features`, `about`, `roadmap`, `blog`, `patch-notes`, `privacy`, `terms`, `shared.tsx` | Implemented UI. Privacy dan terms jelas berupa rancangan. |
| `auth-flow.tsx`, `login`, `register`, `verify-email`, `onboarding`; `masuk`/`daftar` redirect | Interactive mock: email regex, OTP tetap 123456, jeda 650 ms, cooldown awal 30 detik, Google simulasi. Akses langsung ke `/app` tanpa guard. |
| `layout.tsx`, `app/page.tsx`, `app/messenger.tsx`, `messenger.css` | Interactive mock; rail hover/pin dan sidebar mobile. Media query merupakan bukti kode responsif, bukan bukti inspeksi visual. |
| `messenger.tsx`, `read-state.ts` | State React per mount; seed hardcoded, email context di memory, draft per key dan FileReader data URL. Tidak ditemukan localStorage, IndexedDB, network API, session atau websocket; refresh kehilangan perubahan. |
| `app/read-state.test.mjs` | Ada tes helper unread, bukan bukti uji browser/server. |

## Fitur aktual, input dan kebutuhan

| Fitur / tujuan | Status, input, validasi, state dan feedback | Kebutuhan backend dan gap |
|---|---|---|
| Login & register: akses akun | Interactive mock: email regex lokal; Google -> simulasi; OTP tetap; loading dan error palsu. | OTP lifecycle, OIDC, server session, generic errors, cooldown, abuse prevention; tidak boleh mempertahankan OTP tetap. |
| Onboarding: profil pertama | Interactive mock: nama min. 2 karakter; success -> `/app`; nama tidak diteruskan ke messenger. Route dapat dibuka langsung. | Identitas sesi, validasi dan simpan profil, onboarding flag, cek username, retry aman. |
| DM: pilih teman, lihat chat | Interactive mock: Nara/Awan/Sasa tetap; pencarian nama dan isi menggunakan satu variabel; empty state; waktu lokal. | Direktori pengguna, conversation participant ACL, daftar paginated/unread/search. Nama bukan identifier. |
| Pesan: tulis, edit, hapus, reply, pin | Interactive mock: teks kosong ditolak, ID `Date.now()`, edit/hapus berdasar string Kamu, reply menyimpan salinan teks, pin boolean; enter kirim. | ID/waktu server, urutan, idempotency, author/participant ACL, reply FK, tombstone dan conflict. Tidak ada error jaringan. |
| Emoji, reaction, sticker, mention | Partially implemented: reaction satu heart toggle; emoji/stiker disisipkan sebagai teks; mention statis @Nara tanpa resolver/notifikasi. | Reaction per user unik, catalogue/media, mention tervalidasi dan notifikasi; picker bukan bukti sticker backend. |
| Read/unread | Interactive mock: cursor per key di `read-state.ts`; tab terlihat/fokus, dialog dan menu mobile tertutup, tanpa filter, maksimal 24px dari bawah. Nara cursor awal; reset. | Cursor user+conversation dengan sequence server monoton, atomic max dan reconciliation. Scroll bukan bukti server tanpa ack. |
| Workspace | Interactive mock: nama wajib, deskripsi opsional, ID waktu, channel umum dan owner contoh; nama/deskripsi dapat diedit. | Workspace + owner membership + channel dalam transaksi; validasi, izin, audit; identitas visual workspace belum ada. |
| Channel/kategori | Partially implemented: nama di-trim, lowercase, spasi menjadi strip; duplikat lokal ditolak. Rename/delete/category/ACL UI tidak ditemukan. | Channel per workspace, unique slug, kategori, ACL dan aturan delete. |
| Undangan | Partially implemented: buat/cabut teks contoh, salin dan pratinjau berupa notice. Tidak ada join, expiry, kuota atau link nyata. | Token rahasia ter-hash, expiry, revoke, redeem atomik, preview minimum dan membership. |
| Anggota dan role | Partially implemented: daftar seed, dropdown permission/title dan tambah title; UI menyatakan pembatasan belum diterapkan. | Membership, capability, channel ACL, last-owner invariant; title terpisah dari permission. |
| Profil dan media | Interactive mock: nama, username, bio, status, avatar/banner FileReader; data URL di memory tanpa validasi ukuran/tipenya. | Validasi, uniqueness, upload intent, ownership, content validation, scanning, private storage; username bukan identitas internal. |
| Identitas workspace | Planned only: nickname/avatar khusus workspace tidak ditemukan; title anggota baru mock. | Membership profile override, fallback global dan izin edit. |
| Device/session | Interactive mock: daftar contoh, cabut item selain indeks pertama; bukan deteksi perangkat atau sesi. | Sesi server milik user, revoke current/other, metadata device yang tidak dianggap bukti identitas. |
| Settings | Partially implemented: profile/device/preferences dan reset messages/workspaces/drafts/cursor; reset tidak mereset profile/roles/sessions. Notification/privacy/accessibility belum ada. | Settings terversi, notification/privacy options; reset demo tetap lokal, bukan endpoint produksi otomatis. |

## Alur aksi dan pemetaan

Semua mutation: parse -> validasi -> autentikasi -> otorisasi resource -> transaksi -> response dan durable event -> UI update. Rincian field dan respons ada di [API](05-api.md).

| Journey | Alur dan kondisi gagal |
|---|---|
| Login email | Challenge email -> respons generik -> verify code single-use -> sesi -> profil/onboarding. Invalid/expired generik, 429 cooldown, provider failure tanpa kebocoran akun. |
| Google | Start -> state/nonce/PKCE -> callback OIDC terverifikasi -> identity linking aman -> sesi. Pembatalan dan invalid state ditangani; token tidak dikirim ke log. |
| Profil | PATCH profil -> validasi username -> media upload intent -> signed PUT -> finalize verify -> attach. 409 nama, 403 ownership, 422 file invalid. |
| Buka DM | Buat/ambil conversation menurut participant ID -> cek dua pihak -> GET messages cursor. Tanpa bukti partisipasi, jangan ungkap keberadaan data. |
| Tulis/balas/edit/hapus | Membership/ACL -> idempotency -> urutan -> transaksi pesan/reply/tombstone/outbox -> respons -> event. 403/404, 409 revision, 429; optimistic UI harus rollback/reconcile. |
| Reaksi/pin/baca | Cek hak saat mutation; reaction unik, pin sesuai izin; cursor per user monoton hanya setelah melihat pesan -> ack server. Event hanya subscriber berhak. |
| Buat workspace/channel | Actor terverifikasi -> validasi -> workspace, owner, default channel atomik -> event; channel management butuh capability. 409 conflict; rollback bila gagal. |
| Undang/join | Admin membuat token -> preview minimum -> sesi user -> redeem atomik terhadap expiry, revoke dan kuota -> membership. 404/410 generik dan 409 already member menurut policy. |
| Anggota/role | Authority current actor + hierarchy + last-owner check -> transaction + audit. Display title tidak pernah memberi capability. |
| Perangkat/settings | GET/DELETE sesi milik user dan GET/PATCH preferences; revoke current mengakhiri sesi. Metadata perangkat perkiraan saja. |

## Open questions

Apakah DM grup, E2EE, moderasi, retention, private channels, edit/delete window, hak pin, read receipt opt-in, batas attachment dan umur minimum dibutuhkan? E2EE tidak diasumsikan; jika dipilih, search, storage, recovery dan key management perlu desain ulang. UI saat ini belum mengenal auth expiry, server error, optimistic reconciliation, pagination atau reconnect.
