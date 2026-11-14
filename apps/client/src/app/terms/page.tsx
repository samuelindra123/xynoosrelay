import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Ketentuan Layanan",
  description: "Aturan penggunaan akun, profil, pesan, workspace, dan media di XYNOSS Relay.",
};

export default function Terms() {
  return (
    <main className="interior legal-page">
      <section className="wrap legal-hero">
        <p className="overline">INFORMASI / KETENTUAN LAYANAN</p>
        <h1>Ruang bersama,<br /><em>aturan yang jelas.</em></h1>
        <p className="interior-lead">Ketentuan penggunaan akun, percakapan, workspace, dan konten di XYNOSS Relay.</p>
        <p className="legal-updated">Terakhir diperbarui: 14 November 2026 · Berlaku untuk layanan Relay dan situs ini.</p>
      </section>
      <section className="wrap legal-layout">
        <aside className="legal-aside">
          <span>DI HALAMAN INI</span>
          <a href="#cakupan">Cakupan</a>
          <a href="#akun">Akun dan keamanan</a>
          <a href="#perilaku">Penggunaan yang wajar</a>
          <a href="#konten">Konten pengguna</a>
          <a href="#layanan">Ketersediaan layanan</a>
          <a href="#hak">Hak dan penegakan</a>
          <a href="#perubahan">Perubahan ketentuan</a>
        </aside>
        <div className="legal-content">
          <div className="legal-notice">
            <strong>Ketentuan operasional untuk pratinjau; identitas pengelola perlu dilengkapi.</strong>
            <p>Ketentuan ini menjelaskan perilaku aplikasi yang tersedia saat ini. Nama hukum pengelola, alamat resmi, kanal pengaduan, dan mekanisme penyelesaian sengketa belum diumumkan. Dokumen ini harus ditinjau dan dilengkapi sebelum dianggap sebagai perjanjian final.</p>
          </div>

          <section id="cakupan">
            <span>01 / CAKUPAN</span>
            <h2>Layanan Relay</h2>
            <p>Ketentuan ini berlaku ketika kamu mengakses situs atau menggunakan fitur Relay, termasuk masuk dengan email atau Google, mengelola profil dan sesi, bertukar pesan langsung, menggunakan workspace/channel, mengelola undangan, dan mengunggah avatar atau banner. Dengan menggunakan layanan, kamu setuju mengikuti ketentuan ini dan <Link href="/privacy">Kebijakan Privasi</Link>.</p>
          </section>

          <section id="akun">
            <span>02 / AKUN DAN KEAMANAN</span>
            <h2>Jaga akses akunmu</h2>
            <ul>
              <li>Gunakan alamat email dan identitas yang dapat kamu akses. Informasi profil yang kamu tampilkan harus akurat dan tidak menyamar sebagai orang lain.</li>
              <li>Jangan membagikan kode masuk, token, atau akses sesi kepada orang lain. Cabut sesi yang tidak kamu kenali melalui pengaturan.</li>
              <li>Kamu bertanggung jawab atas aktivitas yang dilakukan melalui sesi akunmu sepanjang diwajibkan oleh hukum yang berlaku. Beri tahu pengelola melalui kanal resmi jika kamu menduga akun disalahgunakan.</li>
              <li>Penggunaan layanan tunduk pada batas usia dan persyaratan hukum yang berlaku. Batas usia minimum resmi untuk Relay masih harus ditetapkan pengelola.</li>
            </ul>
          </section>

          <section id="perilaku">
            <span>03 / PENGGUNAAN YANG WAJAR</span>
            <h2>Jaga ruang tetap aman</h2>
            <p>Kamu tidak boleh menggunakan Relay untuk melanggar hukum, mengancam atau melecehkan orang, membagikan konten tanpa hak, mengirim spam, mencoba mengambil alih akun, mengganggu layanan, menghindari kontrol akses workspace/channel, atau mengunggah file berbahaya. Jangan mencoba mengakses pesan, media, sesi, atau workspace yang bukan hakmu.</p>
            <p>Gunakan pengaturan owner workspace dan akses channel sesuai kebutuhan. Undangan bersifat terbatas menurut masa berlaku dan jumlah penggunaan yang ditampilkan pada saat dibuat.</p>
          </section>

          <section id="konten">
            <span>04 / KONTEN PENGGUNA</span>
            <h2>Konten tetap milikmu</h2>
            <p>Kamu mempertahankan hak atas teks dan media yang kamu kirim. Kamu memberi Relay izin terbatas untuk menyimpan, memproses, mengirim, dan menampilkan konten tersebut sejauh diperlukan untuk menjalankan fitur yang kamu gunakan. Izin ini mencakup menampilkan profil kepada pengguna lain dan menyampaikan pesan kepada peserta percakapan atau anggota channel yang memiliki akses.</p>
            <p>Kamu bertanggung jawab memastikan memiliki hak untuk mengunggah atau membagikan konten. Jangan kirim informasi yang tidak ingin dilihat oleh penerima yang memiliki akses. Relay saat ini tidak menjanjikan enkripsi end-to-end untuk pesan.</p>
          </section>

          <section id="layanan">
            <span>05 / KETERSEDIAAN</span>
            <h2>Perubahan dan gangguan</h2>
            <p>Fitur dapat diperbaiki, diubah, dibatasi, atau dihentikan untuk pemeliharaan, keamanan, atau alasan operasional. Relay berupaya menjaga layanan berjalan, tetapi tidak menjanjikan akses tanpa gangguan atau bebas kesalahan. Cadangkan informasi penting di luar layanan.</p>
            <p>Beberapa fitur bergantung pada penyedia eksternal seperti Google, penyedia email, database, object storage, dan hosting. Ketentuan serta ketersediaan penyedia tersebut juga berlaku pada layanan mereka sendiri.</p>
          </section>

          <section id="hak">
            <span>06 / HAK DAN PENEGAKAN</span>
            <h2>Konten, akun, dan tanggung jawab</h2>
            <p>Relay dan identitas mereknya dimiliki oleh pemegang hak terkait. Kode, aset, dan materi situs tidak diberikan lisensi terbuka kecuali dinyatakan terpisah. Jika ada dugaan pelanggaran atau penyalahgunaan, pengelola dapat membatasi sesi atau akses sesuai hukum dan kebutuhan keamanan.</p>
            <p>Ketentuan ini tidak menghapus hak konsumen atau hak wajib lain yang diberikan hukum. Batas tanggung jawab dan prosedur pengaduan harus dirumuskan oleh pengelola setelah identitas dan bentuk badan hukumnya dikonfirmasi.</p>
          </section>

          <section id="perubahan">
            <span>07 / PERUBAHAN KETENTUAN</span>
            <h2>Versi yang berlaku</h2>
            <p>Perubahan ketentuan akan dicatat di halaman ini dengan tanggal pembaruan. Perubahan penting sebaiknya diberitahukan melalui layanan sebelum berlaku. Jika kamu tidak setuju, hentikan penggunaan dan gunakan kendali akun yang tersedia. Penghapusan akun penuh belum tersedia melalui UI saat ini.</p>
            <p>Untuk penjelasan mengenai data, baca <Link href="/privacy">Kebijakan Privasi</Link>. Peraturan terkait konsumen dapat dilihat pada <a href="https://peraturan.bpk.go.id/Details/45288/uu-no-8-tahun-1999">Undang-Undang Nomor 8 Tahun 1999 tentang Perlindungan Konsumen</a>.</p>
          </section>
          <div className="legal-bottom">Dokumen terakhir diperbarui pada 14 November 2026.</div>
        </div>
      </section>
    </main>
  );
}
