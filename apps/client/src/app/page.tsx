const Arrow = ({ diagonal = false }: { diagonal?: boolean }) => <span aria-hidden="true" className="arrow">{diagonal ? "\u2197" : "\u2192"}</span>;

import { Mark } from "./shared";

import { ProductDemo } from "./product-demo";

export default function Home() {
  return (
    <main>
      <section className="hero wrap" id="top">
        <div className="hero-copy">
          <div className="eyebrow"><span className="status-dot" /> SEBUAH CARA BARU UNTUK TERHUBUNG</div>
          <h1>Ngobrol saja.<br /><em>Selebihnya,</em><br />biar ringan<span className="period">.</span></h1>
          <p className="hero-description">Untuk cerita panjang, kabar singkat, dan semua hal di antaranya. Ada ruang yang terasa pas untuk setiap percakapan.</p>
          <div className="hero-actions"><a className="button button-dark" href="#tentang">Jelajahi Relay <Arrow diagonal /></a><span className="action-note">Lebih dekat, tanpa<br />banyak sekat.</span></div>
        </div>
        <ProductDemo />
        <div className="hero-foot"><span>MESSAGING WITHOUT THE WEIGHT.</span><span>GULIR UNTUK MENJELAJAHI <span className="down">&#8595;</span></span></div>
      </section>

      <section className="manifesto" id="tentang">
        <div className="wrap manifesto-inner"><div className="section-label"><span>01 / TENTANG RELAY</span><span className="label-line" /></div><div className="manifesto-copy"><p className="tiny-heading">HAL BAIK DIMULAI DARI SAPAAN.</p><h2>Yang penting bukan<br />aplikasinya. <span>Yang penting<br />adalah ceritanya.</span></h2><div className="manifesto-bottom"><div className="asterisk">&#10035;</div><p>XYNOSS Relay lahir dari gagasan sederhana: teknologi seharusnya memberi ruang untuk lebih dekat, bukan membuatnya terasa rumit.</p></div></div></div>
      </section>

      <section className="principles wrap" id="gagasan">
        <div className="section-label"><span>02 / GAGASAN KAMI</span><span className="label-line" /></div>
        <div className="principles-heading"><div><p className="tiny-heading">SEDIKIT LEBIH SEDERHANA</p><h2>Lebih sedikit distraksi.<br /><em>Lebih banyak koneksi.</em></h2></div><p>Karena rasanya menyenangkan ketika percakapan bisa mengalir begitu saja.</p></div>
        <div className="principle-list">
          <article><span className="number">01</span><div><h3>Ringan rasanya.</h3><p>Ruang yang memberi napas, supaya perhatian tetap pada orang dan cerita di dalamnya.</p></div><span className="principle-icon">&#9676;</span></article>
          <article><span className="number">02</span><div><h3>Sederhana caranya.</h3><p>Hal-hal yang penting dibuat terasa alami. Tak perlu berpikir panjang untuk mulai menyapa.</p></div><span className="principle-icon">&#8599;</span></article>
          <article><span className="number">03</span><div><h3>Dekat maknanya.</h3><p>Dari sapaan pertama sampai obrolan larut malam, setiap momen punya tempatnya.</p></div><span className="principle-icon">&#10035;</span></article>
        </div>
      </section>

      <section className="closing" id="kenalan"><div className="wrap closing-inner"><div className="closing-top"><span>03 / SAMPAI JUMPA DI RELAY</span><Mark /></div><div className="closing-main"><p>UNTUK SETIAP CERITA YANG MENUNGGU DIMULAI</p><h2>Halo, <em>percakapan</em><br />yang lebih ringan<span>.</span></h2><p className="closing-desc">XYNOSS Relay sedang disiapkan. Sampai saatnya tiba, mari beri ruang untuk percakapan yang lebih berarti.</p><a className="button button-light" href="/features">Kenali produknya <Arrow diagonal /></a></div><div className="closing-stamp">LESS APP.<br /><span>MORE</span><br />CONVERSATION.</div></div></section>

    </main>
  );
}
