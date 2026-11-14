import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Catatan pembaruan",
  description: "Riwayat perubahan fitur, kenyamanan, dan keamanan XYNOSS Relay.",
};

const releases = [
  { date: "14 NOV 2026", time: "11.50 WIB", version: "0.4.2", title: "Identitas profil lebih konsisten", kind: "PENGALAMAN", details: ["Avatar profil kini disajikan melalui endpoint media profil agar dapat dimuat oleh pengguna lain pada pesan personal dan channel.", "Penyegaran profil, status kustom, dan ketersediaan diterapkan pada daftar percakapan.", "Dialog crop avatar dan banner memberi pratinjau sebelum media disimpan."] },
  { date: "14 NOV 2026", time: "10.47 WIB", version: "0.4.1", title: "Upload media lebih jelas", kind: "PROFIL", details: ["Validasi tipe dan ukuran file dilakukan sebelum upload.", "Status upload dan kesalahan finalize ditampilkan agar pengguna tahu langkah berikutnya.", "Endpoint media profil hanya menyajikan gambar siap pakai yang masih terpasang pada profil."] },
  { date: "11 NOV 2026", time: "20.52 WIB", version: "0.4.0", title: "Fondasi ruang kerja dan percakapan", kind: "FITUR", details: ["Detail workspace menampilkan anggota dan konteks akses.", "Riwayat percakapan, reaksi, tanda baca, dan kontrol pesan mendapat dukungan UI/API.", "Kontrol sesi, akses channel privat, serta undangan tersedia dari alur aplikasi."] },
  { date: "7 NOV 2026", time: "23.34 WIB", version: "0.3.0", title: "Relay mulai menemukan bentuk", kind: "PRODUK", details: ["Situs produk dan ruang percakapan mulai disatukan dalam satu pengalaman.", "Navigasi dasar untuk masuk, profil, dan percakapan dibangun sebagai fondasi awal."] },
];

export default function PatchNotes() {
  return (
    <main className="interior notes-page">
      <section className="wrap notes-hero">
        <div><p className="overline">RUANG INFORMASI / CATATAN PEMBARUAN</p><h1>Yang baru,<br /><em>dan yang terus dirapikan.</em></h1><p className="interior-lead">Riwayat perubahan Relay, dari fondasi percakapan sampai detail kecil yang membuatnya lebih nyaman.</p></div>
        <div className="notes-number" aria-label="Empat catatan pembaruan"><span>CATATAN</span><strong>04<span>.</span></strong><small>NOVEMBER 2026</small></div>
      </section>
      <section className="wrap notes-content">
        <aside className="notes-rail"><span>ARSIP PEMBARUAN</span><span>VERSI 0.3 — 0.4</span><Link href="/blog">Baca jurnal &#8599;</Link></aside>
        <div className="notes-timeline" aria-label="Riwayat pembaruan">
          {releases.map((release) => (
            <article className="notes-entry" key={release.version}>
              <div className="notes-entry-date"><time>{release.date}</time><span>{release.time}</span></div>
              <div className="notes-entry-content">
                <div className="notes-entry-heading"><p className="overline">{release.kind} / VERSI {release.version}</p><h2>{release.title}</h2></div>
                <ul>{release.details.map((detail) => <li key={detail}>{detail}</li>)}</ul>
              </div>
              <span className="notes-entry-version">{release.version}</span>
            </article>
          ))}
          <p className="notes-disclaimer">Catatan ini merangkum perubahan pada proyek Relay. Ketersediaan fitur dapat bergantung pada konfigurasi deployment.</p>
        </div>
      </section>
    </main>
  );
}
