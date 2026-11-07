# XYNOSS Relay

### Messaging without the weight.

**Less app. More conversation.**

XYNOSS Relay adalah gagasan tentang komunikasi yang lebih ringan: ruang untuk menyapa, berbagi cerita, dan merasa dekat tanpa kerumitan yang tidak perlu. Repositori ini berisi website produk yang memperkenalkan visi, prinsip desain, dan arah pengembangan Relay.

> **Status proyek:** website presentasi produk. Relay masih disiapkan; ilustrasi, demo, dan rencana pengembangan adalah konsep, bukan jaminan fitur atau ketersediaan platform.

---

## Sekilas

Website ini dirancang sebagai pengalaman editorial yang hangat dan responsif. Di halaman utama, demo bergerak memperlihatkan konsep percakapan dalam tampilan ponsel, desktop, dan website. Pengunjung dapat memilih tampilan atau menjeda pergantian animasi; preferensi pengurangan gerak juga dihormati.

| Halaman | Isi |
| --- | --- |
| `/` | Pengenalan Relay dan demo konsep interaktif |
| `/features` | Pengalaman yang dituju dan status kemampuan |
| `/about` | Cerita dan filosofi di balik Relay |
| `/roadmap` | Arah pengembangan tanpa janji tanggal rilis |
| `/blog` | Jurnal dan pemikiran tentang Relay |
| `/patch-notes` | Ruang untuk catatan pembaruan yang diumumkan |
| `/masuk` dan `/daftar` | Informasi bahwa akses akun belum tersedia |
| `/privacy` dan `/terms` | Rancangan awal informasi privasi dan ketentuan |

Halaman privasi dan ketentuan **belum merupakan dokumen resmi**. Identitas pengelola, praktik pemrosesan data, kontak, dan ketentuan layanan perlu dilengkapi serta ditinjau sebelum digunakan untuk peluncuran.

## Teknologi

- **Next.js 16** dengan App Router
- **React 19** dan **TypeScript**
- **Tailwind CSS 4** serta CSS khusus untuk identitas visual dan motion
- **pnpm** sebagai package manager

## Mulai secara lokal

Prasyarat: Node.js yang kompatibel dengan Next.js 16 dan pnpm. Versi package manager yang dicatat proyek adalah `pnpm@12.18.0`.

```bash
cd apps/client
pnpm install
pnpm dev
```

Buka [http://localhost:3000](http://localhost:3000) di browser. Jika port tersebut sedang digunakan, Next.js akan menampilkan alamat aktual di terminal.

## Pemeriksaan kualitas

Jalankan dari `apps/client`:

```bash
pnpm lint
pnpm exec tsc --noEmit
pnpm build
```

Untuk menjalankan build produksi secara lokal:

```bash
pnpm start
```

Jalankan `pnpm build` terlebih dahulu sebelum `pnpm start`.

## Struktur proyek

```text
.
|-- README.md
`-- apps/
    `-- client/
        |-- public/              # Aset statis
        |-- src/app/             # Route App Router, layout, dan komponen bersama
        |-- package.json
        `-- pnpm-lock.yaml
```

## Batas cakupan

Repositori ini berfokus pada website produk. Tombol masuk dan daftar mengarah ke halaman informasi; belum ada autentikasi produksi, backend messenger, atau database yang diimplementasikan. Catatan pembaruan tidak mencantumkan rilis yang belum diumumkan.

---

**XYNOSS Relay** - Messaging without the weight.
