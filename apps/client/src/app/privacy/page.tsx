import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Kebijakan Privasi",
  description: "Cara XYNOSS Relay mengelola informasi akun, profil, pesan, sesi, dan media.",
};

export default function Privacy() {
  return (
    <main className="interior legal-page">
      <section className="wrap legal-hero">
        <p className="overline">INFORMASI / KEBIJAKAN PRIVASI</p>
        <h1>Privasi, dengan<br /><em>lebih banyak kejelasan.</em></h1>
        <p className="interior-lead">Ringkasan data yang diproses Relay, alasan penggunaannya, dan kendali yang tersedia.</p>
        <p className="legal-updated">Terakhir diperbarui: 14 November 2026 · Berlaku untuk layanan Relay dan situs ini.</p>
      </section>
      <section className="wrap legal-layout">
        <aside className="legal-aside">
          <span>DI HALAMAN INI</span>
          <a href="#pengelola">Pengelola</a>
          <a href="#data">Data yang diproses</a>
          <a href="#penggunaan">Penggunaan data</a>
          <a href="#visibilitas">Visibilitas konten</a>
          <a href="#penyedia">Penyedia layanan</a>
          <a href="#retensi">Penyimpanan dan keamanan</a>
          <a href="#kendali">Pilihan dan hak</a>
        </aside>
        <div className="legal-content">
          <div className="legal-notice">
            <strong>Informasi implementasi; rincian pengelola masih perlu dilengkapi.</strong>
            <p>Halaman ini menjelaskan perilaku yang terlihat pada aplikasi saat ini. XYNOSS Relay adalah nama produk; nama hukum pengendali, alamat resmi, kanal permintaan privasi, dan jadwal retensi log belum ditetapkan di repositori. Kebijakan ini belum dapat dianggap sebagai dokumen final sebelum rincian tersebut diumumkan.</p>
          </div>

          <section id="pengelola">
            <span>01 / PENGELOLA</span>
            <h2>Siapa yang mengelola data?</h2>
            <p>Data diproses untuk menjalankan XYNOSS Relay oleh pengelola layanan. Nama orang atau badan hukum, alamat, dan kontak resmi pengendali belum tersedia pada halaman ini. Jangan mengirim data sensitif melalui kanal publik seperti issue repositori. Kanal khusus untuk permintaan privasi harus diumumkan sebelum peluncuran publik.</p>
          </section>

          <section id="data">
            <span>02 / DATA</span>
            <h2>Informasi yang dapat diproses</h2>
            <ul>
              <li><strong>Akun dan masuk:</strong> alamat email, identitas penyedia OAuth, status onboarding, serta data yang diperlukan untuk memulai dan memelihara sesi.</li>
              <li><strong>Profil:</strong> nama tampilan, username, bio, ketersediaan, status kustom, avatar, dan banner yang kamu pilih.</li>
              <li><strong>Komunikasi:</strong> isi pesan langsung dan pesan channel, waktu kirim, revisi, reaksi, penanda baca, keanggotaan workspace, serta informasi undangan dan akses channel.</li>
              <li><strong>Perangkat dan keamanan:</strong> label/perkiraan platform sesi, waktu aktivitas, IP dan user agent dalam log permintaan, serta ID permintaan untuk pemecahan masalah.</li>
              <li><strong>Media:</strong> file gambar yang diunggah, tipe dan ukuran file, status pemrosesan, serta metadata yang diperlukan untuk menampilkan gambar.</li>
              <li><strong>Situs:</strong> data teknis yang diproses browser dan penyedia hosting saat halaman diminta. Detail cookie analitik atau iklan tidak dinyatakan karena fitur tersebut tidak dikonfigurasi dalam kode aplikasi yang ditinjau.</li>
            </ul>
          </section>

          <section id="penggunaan">
            <span>03 / PENGGUNAAN</span>
            <h2>Untuk apa data digunakan?</h2>
            <p>Data digunakan untuk membuat dan mengamankan akun, mengirim kode masuk, menampilkan profil, menyimpan dan mengirim percakapan, mengelola workspace serta hak akses channel, memproses media, menjaga sesi, mencegah penyalahgunaan, dan mendiagnosis gangguan. Dasar pemrosesan yang berlaku untuk setiap tujuan harus ditetapkan oleh pengelola berdasarkan konteks layanan dan hukum yang berlaku.</p>
          </section>

          <section id="visibilitas">
            <span>04 / VISIBILITAS</span>
            <h2>Siapa yang dapat melihat konten?</h2>
            <p>Username, nama tampilan, bio, ketersediaan, status kustom, avatar, dan banner dapat ditampilkan pada pencarian pengguna dan kartu profil. Pesan langsung tersedia bagi peserta percakapan. Pesan workspace tersedia bagi anggota yang memiliki akses ke channel; channel privat memiliki pemeriksaan akses tersendiri. Jangan menaruh kata sandi, kode rahasia, atau informasi yang tidak ingin dibagikan di profil atau percakapan.</p>
          </section>

          <section id="penyedia">
            <span>05 / PENYEDIA</span>
            <h2>Layanan pihak ketiga</h2>
            <p>Konfigurasi proyek dapat menggunakan Google untuk OAuth, Neon/PostgreSQL untuk akun dan data aplikasi, Neon Object Storage atau penyedia S3-compatible untuk media, Resend atau SMTP untuk email, dan Vercel untuk hosting. Penyedia aktif dan lokasi pemrosesan dapat berbeda menurut environment. Penyedia tersebut menerima data hanya sejauh diperlukan untuk fungsi yang dikonfigurasi. Data dapat diproses di luar Indonesia sesuai lokasi layanan yang dipilih; lokasi produksi harus dikonfirmasi dan dicantumkan oleh pengelola sebelum peluncuran.</p>
          </section>

          <section id="retensi">
            <span>06 / RETENSI DAN KEAMANAN</span>
            <h2>Berapa lama data disimpan?</h2>
            <p>Profil dan pesan disimpan di database sampai diubah atau dihapus melalui fitur yang tersedia atau sampai ada kebijakan penghapusan yang berlaku. Sesi memiliki masa berlaku dan dapat dicabut dari pengaturan. Avatar/banner dapat dilepas; penggantian atau penghapusan menandai media lama untuk dihapus dari storage. Log operasional mencatat metadata permintaan, bukan isi body; lama penyimpanannya ditentukan oleh konfigurasi hosting dan belum ditetapkan di proyek.</p>
            <p>Aplikasi menggunakan session cookie HttpOnly dan Secure pada production, pemeriksaan CSRF untuk aksi yang mengubah data, pembatasan format/ukuran media, serta kontrol akses untuk ruang privat. Pesan tidak dinyatakan sebagai end-to-end encrypted; jangan menganggap isi pesan hanya dapat dibaca di perangkat pengirim dan penerima.</p>
            <p>Penghapusan akun penuh dan jadwal retensi formal belum tersedia melalui UI. Pengelola harus menetapkan dan mengumumkan prosesnya sebelum layanan dianggap siap diluncurkan secara luas.</p>
          </section>

          <section id="kendali">
            <span>07 / KENDALI DAN HAK</span>
            <h2>Pilihan yang tersedia</h2>
            <p>Kamu dapat memperbarui bio dan status, mengganti atau menghapus avatar/banner, menghapus pesan milikmu, serta mencabut sesi aktif melalui pengaturan yang tersedia. Permintaan akses, koreksi, penghapusan, keberatan, atau pertanyaan privasi membutuhkan kanal resmi pengelola yang belum dicantumkan; hindari menyertakan data sensitif di forum publik.</p>
            <p>Informasi ini disusun dengan merujuk pada <a href="https://peraturan.bpk.go.id/Details/229798/uu-no-27-tahun-2022">Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi</a> dan <a href="https://peraturan.bpk.go.id/Details/122030/pp-no-71-tahun-2019">Peraturan Pemerintah Nomor 71 Tahun 2019 tentang Penyelenggaraan Sistem dan Transaksi Elektronik</a>. Rujukan tersebut bukan pengganti peninjauan hukum atas praktik operasional Relay.</p>
          </section>

          <section id="perubahan">
            <span>08 / PERUBAHAN</span>
            <h2>Pembaruan kebijakan</h2>
            <p>Jika praktik data berubah, halaman dan tanggal pembaruan ini akan diperbarui. Perubahan penting perlu diberitahukan melalui kanal layanan yang tersedia. Untuk syarat penggunaan, baca <Link href="/terms">Ketentuan Layanan</Link>.</p>
          </section>
          <div className="legal-bottom">Dokumen terakhir diperbarui pada 14 November 2026.</div>
        </div>
      </section>
    </main>
  );
}
