import type { Metadata } from "next";
import Link from "next/link";
export const metadata: Metadata = {
  title: "Ketentuan",
  description: "Informasi awal mengenai ketentuan penggunaan XYNOSS Relay.",
};
export default function Terms() {
  return (
    <main className="interior legal-page">
      <section className="wrap legal-hero">
        <p className="overline">INFORMASI / KETENTUAN</p>
        <h1>
          Jelas dari
          <br />
          <em>awal.</em>
        </h1>
        <p className="interior-lead">
          Hal-hal yang perlu kamu ketahui tentang situs ini dan status layanan
          Relay.
        </p>
      </section>
      <section className="wrap legal-layout">
        <aside className="legal-aside">
          <span>DI HALAMAN INI</span>
          <a href="#status">Status dokumen</a>
          <a href="#situs">Tentang situs</a>
          <a href="#akses">Akses Relay</a>
          <a href="#perubahan">Perubahan informasi</a>
        </aside>
        <div className="legal-content">
          <div className="legal-notice">
            <strong>Rancangan awal, belum menjadi ketentuan resmi.</strong>
            <p>
              Identitas pengelola, cakupan layanan, mekanisme penyelesaian
              sengketa, dan kontak belum ditetapkan dalam proyek ini. Dokumen
              resmi perlu dilengkapi dan ditinjau sebelum layanan diluncurkan.
            </p>
          </div>
          <section id="status">
            <span>01 / STATUS</span>
            <h2>Tentang halaman ini</h2>
            <p>
              Halaman ini memberikan informasi awal tentang situs XYNOSS Relay.
              Halaman ini belum merupakan syarat penggunaan lengkap untuk
              layanan messenger yang masih disiapkan.
            </p>
          </section>
          <section id="situs">
            <span>02 / SITUS</span>
            <h2>Informasi dan ilustrasi</h2>
            <p>
              Situs ini memperkenalkan visi dan arah Relay. Ilustrasi produk,
              demo, dan rencana pengembangan adalah konsep; detailnya dapat
              berubah dan tidak dengan sendirinya menunjukkan ketersediaan
              fitur.
            </p>
          </section>
          <section id="akses">
            <span>03 / AKSES</span>
            <h2>Akun dan layanan</h2>
            <p>
              Masuk dan pendaftaran belum tersedia. Jika layanan dibuka,
              ketentuan resmi perlu menjelaskan kelayakan, aturan penggunaan,
              tanggung jawab pengguna dan pengelola, serta cara menghubungi
              pengelola.
            </p>
          </section>
          <section id="perubahan">
            <span>04 / PERUBAHAN</span>
            <h2>Informasi dapat berkembang</h2>
            <p>
              Informasi di situs dapat berubah seiring pengembangan. Versi
              ketentuan resmi dan cara pemberitahuan perubahannya akan perlu
              ditetapkan sebelum layanan digunakan.
            </p>
          </section>
          <div className="legal-bottom">
            Baca juga <Link href="/privacy">Informasi privasi &#8599;</Link>
          </div>
        </div>
      </section>
    </main>
  );
}
