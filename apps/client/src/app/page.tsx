import Link from "next/link";
import { ProductDemo } from "./product-demo";
import { Signal } from "./signal";
export default function Home() {
  return (
    <main className="home-page">
      <section className="home-hero wrap" id="top">
        <div className="hero-heading">
          <p className="eyebrow">
            <span className="status-dot" /> SEDANG DISIAPKAN. UNTUK CERITAMU.
          </p>
          <h1>
            Kurangi ribet.
            <br />
            Tambah{" "}
            <span>
              dekat
              <svg viewBox="0 0 600 36" aria-hidden="true">
                <path d="M8 22Q260 -4 590 16" />
              </svg>
            </span>
            <i>.</i>
          </h1>
          <div className="hero-intro">
            <p>
              Obrolan panjang. Kabar singkat. Atau sekadar,
              <br className="desktop-break" /> &ldquo;hei, kamu apa
              kabar?&rdquo;
              <br />
              Ada ruang untuk semuanya.
            </p>
            <Link className="button button-dark" href="/features">
              Kenalan dengan Relay <span>&#8599;</span>
            </Link>
          </div>
        </div>
        <div className="hero-showcase">
          <div className="hero-side-label">
            LESS APP.
            <br />
            MORE CONVERSATION.
          </div>
          <ProductDemo />
          <div className="hero-sticker" aria-hidden="true">
            halo<span>&#10035;</span>
          </div>
        </div>
        <div className="hero-baseline">
          <span>MESSAGING WITHOUT THE WEIGHT.</span>
          <a href="#cerita">ADA CERITA DI BAWAH &#8595;</a>
        </div>
      </section>
      <section className="story-panel wrap" id="cerita">
        <div className="story-visual">
          <Signal />
          <span className="visual-caption">
            SATU SAPAAN BISA MEMBAWA KITA JAUH.
          </span>
        </div>
        <div className="story-copy">
          <p className="overline">01 / RUANG UNTUK YANG BERARTI</p>
          <h2>
            Bukan sibuk
            <br />
            di aplikasinya.
            <br />
            <em>Asyik di ceritanya.</em>
          </h2>
          <p>
            Kami membayangkan messenger yang tahu kapan harus hadir, dan kapan
            harus memberi ruang. Supaya perhatianmu tetap pada orang di seberang
            layar.
          </p>
          <Link className="text-link" href="/about">
            Ada alasan di baliknya <span>&#8599;</span>
          </Link>
        </div>
      </section>
      <section className="home-statement">
        <div className="wrap">
          <p className="overline">02 / GAGASAN YANG MENUNTUN KAMI</p>
          <h2>
            Yang ringan
            <br />
            bisa terasa
            <br />
            <span>begitu berarti.</span>
          </h2>
          <div className="statement-bottom">
            <span className="statement-star" aria-hidden="true">
              &#10035;
            </span>
            <p>
              Sederhana caranya. Dekat maknanya.
              <br />
              Itulah arah yang ingin kami bawa ke setiap keputusan tentang
              Relay.
            </p>
            <Link
              className="round-link"
              href="/roadmap"
              aria-label="Lihat arah pengembangan"
            >
              &#8599;
            </Link>
          </div>
        </div>
      </section>
      <section className="home-reading wrap">
        <div className="reading-heading">
          <p className="overline">03 / DARI BALIK LAYAR</p>
          <h2>
            Masih dirangkai.
            <br />
            <em>Sudah punya cerita.</em>
          </h2>
        </div>
        <div className="reading-links">
          <Link href="/blog">
            <span>JURNAL / 01</span>
            <h3>
              Untuk apa semua ini
              <br />
              dibuat lebih sederhana?
            </h3>
            <b aria-hidden="true">&#8599;</b>
          </Link>
          <Link href="/patch-notes">
            <span>CATATAN PEMBARUAN</span>
            <h3>
              Langkah kecil berikutnya.
              <br />
              Sewaktu sudah siap.
            </h3>
            <b aria-hidden="true">&#8599;</b>
          </Link>
        </div>
      </section>
    </main>
  );
}
