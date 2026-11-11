# XYNOSS Relay — rancangan backend

> **Status: proposal, belum diimplementasikan.** Audit berdasarkan source lokal pada 8 November 2026, termasuk perubahan yang belum di-commit. Dokumen ini bukan bukti deployment atau fitur produksi.

Frontend Next.js direncanakan berkomunikasi melalui HTTPS JSON dengan backend Go modular monolith yang terpisah. Backend menggunakan Chi, pgx, sqlc, goose, Neon PostgreSQL, REST/OpenAPI, Google OAuth dan email OTP. Vercel Functions adalah target deployment backend terpisah; realtime melalui adapter yang dapat diganti. Keputusan ini tidak berarti komponennya sudah tersedia.

## Urutan membaca

1. [Audit frontend dan alur](01-audit-frontend.md)
2. [Arsitektur dan keputusan](02-arsitektur.md)
3. [Modul backend](03-modul.md)
4. [Model data](04-data.md)
5. [Kontrak REST](05-api.md)
6. [Autentikasi dan keamanan](06-keamanan.md)
7. [Realtime dan proof of concept](07-realtime.md)
8. [Infrastruktur dan operasi](08-infrastruktur.md)
9. [Standar engineering](09-standar.md)
10. [Roadmap dan acceptance criteria](10-roadmap.md)

## Risiko utama

- Go runtime Vercel berstatus beta. WebSocket Functions diumumkan public beta dengan contoh Node.js; kompatibilitas upgrade WebSocket pada Go/Chi perlu PoC. Halaman limits Vercel juga bertentangan dengan changelog yang lebih baru: lihat [realtime](07-realtime.md).
- Vercel Hobby ditujukan untuk penggunaan personal/nonkomersial; kesesuaian produk komersial harus ditinjau sebelum rilis. [Dokumentasi Hobby](https://vercel.com/docs/plans/hobby).
- Frontend dan API berbeda origin: cookie, CORS, CSRF, OAuth callback dan browser third-party cookie membutuhkan keputusan domain dan pengujian nyata.
- Target Rp0 tidak menjamin biaya nol untuk produksi atau pertumbuhan. Email, database, storage, realtime, trafik dan operasional memiliki kuota dan risiko biaya.
- OTP tetap, pengguna, undangan dan pesan saat ini adalah simulasi React; tidak ada sesi atau otorisasi backend. [Audit](01-audit-frontend.md) menjelaskan batasnya.

**Fakta** berasal dari source atau dokumentasi resmi yang ditautkan. **Usulan** perlu persetujuan dan validasi. **PoC** berarti belum dibuktikan pada deployment proyek. Tidak ada infrastruktur atau backend yang dibuat oleh pekerjaan dokumentasi ini.
