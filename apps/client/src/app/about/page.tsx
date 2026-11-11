import type { Metadata } from "next";
import { PageEnd } from "../shared";
import { Signal } from "../signal";
export const metadata: Metadata = {
  title: "Cerita kami",
  description: "Alasan dan filosofi di balik XYNOSS Relay.",
};
export default function About() {
  return (
    <main className="interior about-page">
      <section className="wrap about-hero">
        <p className="overline">02 / CERITA KAMI</p>
        <h1>
          Ada banyak cara
          <br />
          untuk <em>terhubung.</em>
          <br />
          Kami memilih yang
          <br />
          terasa manusiawi<span>.</span>
        </h1>
        <div className="about-art">
          <Signal />
        </div>
        <div className="about-hero-foot">
          <span>LESS APP. MORE CONVERSATION.</span>
          <span>GULIR UNTUK MEMBACA &#8595;</span>
        </div>
      </section>
      <section className="about-letter">
        <div className="wrap letter-grid">
          <div className="section-label">
            SEBUAH CATATAN <span className="label-line" />
          </div>
          <div>
            <p className="letter-opening">Halo,</p>
            <h2>
              Kadang yang kita butuhkan hanya tempat untuk berkata,{" "}
              <em>&ldquo;aku di sini.&rdquo;</em>
            </h2>
            <div className="letter-body">
              <p>
                Percakapan tidak selalu dimulai dengan sesuatu yang besar.
                Kadang hanya kabar singkat, pertanyaan kecil, atau cerita yang
                muncul di tengah hari.
              </p>
              <p>
                XYNOSS Relay berangkat dari keinginan untuk memberi momen-momen
                itu ruang yang lebih sederhana. Teknologi boleh bekerja keras di
                belakang layar; percakapan di depannya seharusnya terasa ringan.
              </p>
              <p>
                Kami ingin membuat pengalaman yang memberi perhatian kembali
                kepada orang di seberang layar. Dengan lebih sedikit gangguan,
                ada lebih banyak ruang untuk mendengar.
              </p>
            </div>
            <span className="letter-sign">
              Dengan hangat,
              <br />
              <strong>XYNOSS Relay &#10035;</strong>
            </span>
          </div>
        </div>
      </section>
      <section className="wrap belief-section">
        <p className="overline">YANG MENUNTUN KAMI</p>
        <div className="belief-grid">
          <h2>
            Kesederhanaan
            <br />
            adalah <em>perhatian.</em>
          </h2>
          <div>
            <p>
              Bagi kami, sederhana bukan berarti kehilangan karakter. Sederhana
              berarti memilih dengan saksama apa yang perlu hadir, dan memberi
              ruang pada yang benar-benar penting.
            </p>
            <p>
              Itulah visi yang ingin kami bawa ke setiap keputusan: komunikasi
              yang terasa dekat, jelas, dan menyenangkan.
            </p>
          </div>
        </div>
      </section>
      <PageEnd
        eyebrow="GAGASAN YANG DIBAGIKAN"
        title="Hal baik tumbuh bersama."
        href="/blog"
        link="Baca jurnal Relay"
      />
    </main>
  );
}
