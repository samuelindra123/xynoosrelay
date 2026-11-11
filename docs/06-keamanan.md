# Autentikasi, sesi dan keamanan

[Indeks](README.md) | [API](05-api.md) | [Data](04-data.md)

Semua nilai numerik adalah **usulan awal**, bukan klaim bahwa fitur ada. Email OTP mengandalkan keamanan mailbox dan bukan MFA. Kebijakan risiko dan legal harus disetujui sebelum rilis.

## OTP email

1. `POST /auth/otp/challenges`: validasi bentuk dan panjang email, canonicalization hati-hati (jangan ubah arti alamat provider), batas IP + email-hash + device/risk dan global. Respons 202 generik untuk akun ada/tidak; tidak menjamin pengiriman. Jangan menciptakan sesi.
2. Buat 6 digit CSPRNG seragam; TTL usulan 10 menit, maksimal 5 percobaan per challenge, resend cooldown 60 detik plus kuota per alamat/IP; semuanya dikonfigurasi server dan dinilai threat model. Resend membuat challenge baru, invalidasi sebelumnya atomik; cooldown tidak di-reset dengan refresh.
3. Simpan HMAC berkunci terpisah untuk kode + challenge ID, bukan plaintext atau hash tanpa pepper (enam digit mudah brute force bila DB bocor). Kunci di secret manager/env, key version untuk rotasi; tidak di DB/log/trace/analytics. Bandingkan constant-time. Rate limit distribusi tidak boleh hanya in-memory.
4. Verifikasi dengan row lock: validasi expiry/attempt/state, increment attempt atomik bahkan gagal, sukses consume sekali dan create/link identity + session satu transaksi. Dua verify bersamaan hanya satu sukses. Email delivery eksternal di luar transaksi; kegagalan delivery harus observable tanpa enumeration.
5. Cleanup challenge expired; minimalkan data dan masa simpan. Jangan log raw email bila tidak perlu; gunakan keyed hash untuk korelasi. [OWASP OTP handling](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html).

## Google OAuth / OIDC

Authorization code flow dengan PKCE S256, state anti-CSRF, nonce, exact registered redirect URI; callback backend mengikat state ke browser pre-auth dan menghabiskannya sekali. Verifikasi signature/JWKS, issuer, audience, expiry, nonce serta `email_verified` jika email digunakan; identity unik provider+sub. Jangan gunakan email sebagai satu-satunya stable identifier. Library OIDC matang dipilih saat implementasi. [Google OIDC](https://developers.google.com/identity/openid-connect/openid-connect).

Link akun existing hanya setelah bukti kontrol kedua identitas dan re-auth; jangan menggabungkan otomatis hanya berdasarkan email sama. Hindari open redirect: return target allowlist path relatif. Recovery bila mailbox/provider hilang perlu kebijakan dan dukungan aman; jangan gunakan security questions. Status onboarding disimpan server, bukan diambil dari URL.

## Sesi dan cookie

Secret sesi acak minimal 256 bit usulan, lookup identifier terpisah + verifier hash; constant-time verify; expiry idle/absolute ditegakkan server, revoke immediate dan session rotate setelah autentikasi/perubahan sensitif. Simpan secret hanya di cookie host-only `__Host-` pada API HTTPS dengan Secure, HttpOnly, Path=/, tanpa Domain; SameSite=Lax bila frontend/API same-site, PoC callback OAuth dan browser. Cross-site memerlukan SameSite=None; Secure dan tetap dapat diblokir third-party cookies. Jangan mengasumsikan domain deployment otomatis same-site. Sesi current diidentifikasi server; device/UA sekadar informasi. Logout revoke server dan hapus cookie. [OWASP sessions](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html).

Credentialed CORS hanya origin eksplisit, `Access-Control-Allow-Credentials: true`, tidak wildcard, allow methods/headers minimum dan `Vary: Origin`. Browser `credentials: include`; middleware CSRF double-submit token atau synchronizer token terikat sesi plus Origin check untuk seluruh mutation termasuk login/OTP; tidak mengandalkan CORS/SameSite saja. Non-browser clients butuh skema terpisah yang belum diputuskan.

## Otorisasi, audit, abuse

- Middleware melakukan authentication, service mengecek actor aktif, workspace membership dan effective capability/channel ACL pada setiap read/write/search dan saat subscribe; update sensitif pada transaksi yang sama. Default deny; error 403/404 sesuai kebijakan anti-enumerasi. Jangan menerima role dari client.
- Rate limit terdistribusi untuk OTP request/verify/resend, login, invite preview/redeem, search, send, media dan abuse IP/user/workspace. 429 dengan Retry-After; desain store atomik dan fail-closed untuk auth ketika store gagal. Pertimbangkan challenge adaptif tanpa membocorkan status akun.
- Audit: actor, action, target, outcome, request_id dan timestamp untuk login, revocation, role, invite, moderation dan media. Jangan log OTP, token, cookie, email mentah secara default, isi pesan, presigned URL atau credentials.
- Gunakan TLS, least privilege database dan provider, secret rotation, parameterized sqlc, dependency scan, payload/URL allowlist dan size limits; anti-XSS dengan React escaping dan CSP. Enkripsi in transit/at rest bergantung vendor; **tidak menyatakan E2EE**.
- Media: upload ke object key acak, izin minimum, verifikasi owner, content sniff, ukuran, decode/re-encode gambar bila sesuai, antivirus/quarantine sebelum ready, hapus EXIF jika kebijakan menyatakan. Upload langsung tetap butuh batas ukuran yang ditegakkan storage/gateway; presigned PUT semata bukan jaminan.

## Belum diputuskan

Retensi, recovery, perubahan email dan re-auth, session TTL final, kebutuhan MFA untuk admin, aturan privasi read receipts, anti-abuse threshold, IP storage dan prosedur insiden. Legal review wajib sebelum kebijakan publik dipakai sebagai dokumen resmi.
