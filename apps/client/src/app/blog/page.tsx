import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Jurnal",
  description: "Catatan desain, produk, dan proses membangun XYNOSS Relay.",
};

const entries = [
  {
    date: "14 November 2026",
    time: "11.50 WIB",
    category: "DI BALIK PRODUK / PROFIL",
    title: "Identitas yang ikut hadir di setiap percakapan",
    deck: "Avatar, status, dan bio membantu pesan terasa datang dari seseorang, bukan sekadar akun.",
    body: "Profil yang mudah dikenali membuat ruang percakapan terasa lebih personal. Karena itu kami merapikan cara Relay menampilkan identitas yang sama di kartu profil, daftar percakapan, pesan langsung, dan channel workspace. Perubahan foto profil kini perlu tercermin konsisten di tempat orang benar-benar berbicara.",
  },
  {
    date: "11 November 2026",
    time: "20.52 WIB",
    category: "CATATAN TEKNIS / RUANG KERJA",
    title: "Membuat ruang kerja lebih teratur",
    deck: "Percakapan personal dan channel bersama punya kebutuhan yang berbeda.",
    body: "Relay berkembang dari obrolan langsung menjadi ruang kerja dengan anggota, peran, channel, dan undangan. Fokus kami adalah menjaga navigasi tetap mudah dipahami sambil menambahkan akses yang sesuai untuk pemilik ruang dan anggota.",
  },
  {
    date: "7 November 2026",
    time: "23.34 WIB",
    category: "CATATAN PERTAMA / FILOSOFI",
    title: "Untuk apa semua ini dibuat lebih sederhana?",
    deck: "Karena sebuah sapaan tidak seharusnya terasa seperti pekerjaan.",
    body: "Kita sering mengingat isi percakapan, bukan tombol yang ditekan untuk memulainya. Kabar baik dari teman lama. Pesan singkat di hari yang panjang. Cerita yang tanpa terasa berlangsung sampai malam. Kesederhanaan adalah cara untuk sampai pada sesuatu yang lebih penting: merasa didengar, merasa dekat, dan punya tempat untuk kembali bercerita.",
  },
];

export default function Blog() {
  return (
    <main className="interior journal-page">
      <section className="wrap journal-hero">
        <div>
          <p className="overline">RUANG INFORMASI / JURNAL</p>
          <h1>Catatan tentang<br /><em>menjadi dekat.</em></h1>
          <p className="interior-lead">Pemikiran, keputusan desain, dan cerita proses membangun ruang percakapan Relay.</p>
        </div>
        <div className="journal-seal" aria-hidden="true"><span>&#10035;</span><p>SEBUAH RUANG<br />UNTUK BERCERITA</p></div>
      </section>
      <section className="journal-feature">
        <div className="wrap journal-timeline">
          <div className="journal-index"><span>JURNAL RELAY</span><span>03 CATATAN</span></div>
          <div className="journal-entries">
            {entries.map((entry, index) => (
              <article className="journal-entry" key={entry.title}>
                <div className="journal-entry-meta"><time>{entry.date}</time><span>{entry.time}</span></div>
                <div>
                  <p className="overline">{entry.category}</p>
                  <h2>{entry.title}</h2>
                  <p className="journal-deck">{entry.deck}</p>
                  <p className="journal-body">{entry.body}</p>
                  <div className="journal-sign">XYNOSS Relay <span>&#10035;</span></div>
                </div>
                <span className="journal-entry-number">{String(index + 1).padStart(2, "0")}</span>
              </article>
            ))}
          </div>
        </div>
      </section>
      <section className="wrap journal-next">
        <div><p className="overline">PERJALANAN YANG BERLANJUT</p><h2>Setiap perubahan<br /><em>punya ceritanya.</em></h2></div>
        <div><p>Ikuti catatan produk untuk melihat perubahan yang sudah hadir di Relay.</p><Link className="text-link" href="/patch-notes">Lihat catatan pembaruan &#8599;</Link></div>
      </section>
    </main>
  );
}
