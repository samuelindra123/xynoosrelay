import type { Metadata } from "next";
import Link from "next/link";
export const metadata: Metadata = {
  title: "Privasi",
  description: "Informasi awal mengenai privasi XYNOSS Relay.",
};
export default function Privacy() {
  return (
    <main className="interior legal-page">
      <section className="wrap legal-hero">
        <p className="overline">INFORMASI / PRIVASI</p>
        <h1>
          Privasi itu
          <br />
          <em>penting.</em>
        </h1>
        <p className="interior-lead">
          Kejelasan adalah awal dari rasa percaya. Berikut informasi awal
          mengenai privasi di Relay.
        </p>
      </section>
      <section className="wrap legal-layout">
        <aside className="legal-aside">
          <span>DI HALAMAN INI</span>
          <a href="#status">Status dokumen</a>
          <a href="#situs">Situs ini</a>
          <a href="#layanan">Layanan Relay</a>
          <a href="#hak">Hak dan informasi</a>
        </aside>
        <div className="legal-content">
          <div className="legal-notice">
            <strong>Rancangan awal, belum menjadi kebijakan resmi.</strong>
            <p>
              Identitas pengelola, penyedia layanan, praktik pemrosesan data,
              dan kontak privasi belum dikonfirmasi. Informasi ini perlu
              dilengkapi dan ditinjau sebelum Relay diluncurkan.
            </p>
          </div>
          <section id="status">
            <span>01 / STATUS</span>
            <h2>Tentang halaman ini</h2>
            <p>
              XYNOSS Relay masih disiapkan. Halaman ini menjelaskan batas
              informasi yang tersedia saat ini dan belum menggantikan kebijakan
              privasi resmi yang lengkap.
            </p>
          </section>
          <section id="situs">
            <span>02 / SITUS</span>
            <h2>Informasi saat mengunjungi situs</h2>
            <p>
              Situs ini tidak menyediakan formulir pendaftaran atau masuk yang
              aktif. Infrastruktur hosting atau jaringan yang digunakan untuk
              menyajikan situs dapat memproses data teknis; rincian penyedia,
              log, cookie, dan masa penyimpanannya belum dikonfirmasi.
            </p>
          </section>
          <section id="layanan">
            <span>03 / LAYANAN</span>
            <h2>Ketika Relay tersedia</h2>
            <p>
              Informasi mengenai data akun, pesan, perangkat, tujuan penggunaan,
              dasar pemrosesan, penerima data, masa penyimpanan, keamanan, dan
              penghapusan akan perlu dijelaskan dalam kebijakan resmi sesuai
              praktik layanan yang sebenarnya.
            </p>
          </section>
          <section id="hak">
            <span>04 / HAK DAN INFORMASI</span>
            <h2>Mengetahui pilihanmu</h2>
            <p>
              Hak terkait data pribadi dan cara mengajukan permintaan perlu
              dijelaskan dalam kebijakan resmi, bersama identitas pengendali dan
              kanal kontak yang dapat digunakan. Detail tersebut belum tersedia
              untuk diumumkan.
            </p>
            <p>
              Di Indonesia,{" "}
              <a href="https://peraturan.bpk.go.id/Details/229798/uu-no-27-tahun-2022">
                Undang-Undang Pelindungan Data Pribadi
              </a>{" "}
              mengatur antara lain hak subjek data serta kewajiban pengendali
              dan prosesor data pribadi.
            </p>
          </section>
          <div className="legal-bottom">
            Baca juga <Link href="/terms">Ketentuan penggunaan &#8599;</Link>
          </div>
        </div>
      </section>
    </main>
  );
}
