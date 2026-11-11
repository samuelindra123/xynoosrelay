import Link from "next/link";

export function Mark() {
  return (
    <span className="mark" aria-hidden="true">
      <span />
      <span />
      <span />
      <span />
    </span>
  );
}
const mainLinks = [
  { href: "/features", label: "Produk" },
  { href: "/about", label: "Cerita kami" },
  { href: "/roadmap", label: "Arah pengembangan" },
];
const resources = [
  {
    href: "/blog",
    label: "Jurnal",
    description: "Cerita dan pikiran di balik Relay",
  },
  {
    href: "/patch-notes",
    label: "Catatan pembaruan",
    description: "Ikuti perubahan yang diumumkan",
  },
];
export function Header() {
  return (
    <header className="site-header">
      <div className="header wrap">
        <Link className="brand" href="/" aria-label="XYNOSS Relay, beranda">
          <Mark />
          <span>
            XYNOSS <strong>Relay</strong>
          </span>
        </Link>
        <nav className="desktop-nav" aria-label="Navigasi utama">
          {mainLinks.map((link) => (
            <Link href={link.href} key={link.href}>
              {link.label}
            </Link>
          ))}
          <details className="resources-menu">
            <summary>
              Ruang informasi <span aria-hidden="true">&#8964;</span>
            </summary>
            <div className="resources-popover">
              <p>JELAJAHI RELAY</p>
              {resources.map((link) => (
                <a href={link.href} key={link.href}>
                  <span>
                    <strong>{link.label}</strong>
                    <small>{link.description}</small>
                  </span>
                  <span aria-hidden="true">&#8599;</span>
                </a>
              ))}
            </div>
          </details>
        </nav>
        <div className="header-actions">
          <Link className="login-link" href="/login">
            Masuk <span aria-hidden="true">&#8599;</span>
          </Link>
          <Link className="signup-link" href="/register">
            Daftar <span aria-hidden="true">&#8599;</span>
          </Link>
        </div>
        <details className="mobile-menu">
          <summary aria-label="Buka menu navigasi">
            <span></span>
            <span></span>
          </summary>
          <div className="mobile-panel">
            <nav aria-label="Navigasi seluler">
              {mainLinks.map((link) => (
                <a href={link.href} key={link.href}>
                  {link.label} <span aria-hidden="true">&#8599;</span>
                </a>
              ))}
              <span className="mobile-nav-label">RUANG INFORMASI</span>
              {resources.map((link) => (
                <a href={link.href} key={link.href}>
                  {link.label} <span aria-hidden="true">&#8599;</span>
                </a>
              ))}
            </nav>
            <div className="mobile-actions">
              <a href="/login">Masuk &#8599;</a>
              <a href="/register">Daftar &#8599;</a>
            </div>
          </div>
        </details>
      </div>
    </header>
  );
}
export function Footer() {
  return (
    <footer className="site-footer">
      <div className="wrap footer-invitation">
        <div>
          <p className="overline">SETIAP CERITA PUNYA AWAL</p>
          <h2>
            Mulai dari <em>halo.</em>
          </h2>
        </div>
        <Link className="round-link" href="/features" aria-label="Kenali Relay">
          &#8599;
        </Link>
      </div>
      <div className="wrap footer-grid">
        <div className="footer-brand">
          <Link className="brand" href="/">
            <Mark />
            <span>
              XYNOSS <strong>Relay</strong>
            </span>
          </Link>
          <p>Messaging without the weight.</p>
          <span className="footer-status">
            <i /> Relay sedang disiapkan
          </span>
        </div>
        <div>
          <p>KENALI RELAY</p>
          <nav aria-label="Produk di footer">
            {mainLinks.map((link) => (
              <Link key={link.href} href={link.href}>
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
        <div>
          <p>RUANG INFORMASI</p>
          <nav aria-label="Informasi di footer">
            {resources.map((link) => (
              <Link key={link.href} href={link.href}>
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
        <div>
          <p>HAL-HAL PENTING</p>
          <nav aria-label="Kebijakan di footer">
            <Link href="/privacy">Privasi</Link>
            <Link href="/terms">Ketentuan</Link>
          </nav>
        </div>
      </div>
      <div className="footer-wordmark" aria-hidden="true">
        <span>relay</span>
        <i>&#10035;</i>
      </div>
      <div className="wrap footer-bottom">
        <span>&#169; 2026 XYNOSS RELAY</span>
        <span>LESS APP. MORE CONVERSATION.</span>
        <a href="#main-content">KEMBALI KE ATAS &#8593;</a>
      </div>
    </footer>
  );
}
export function PageEnd({
  eyebrow,
  title,
  href,
  link,
}: {
  eyebrow: string;
  title: string;
  href: string;
  link: string;
}) {
  return (
    <section className="page-end">
      <div className="wrap page-end-inner">
        <div>
          <p className="tiny-heading">{eyebrow}</p>
          <h2>{title}</h2>
        </div>
        <Link className="button button-dark" href={href}>
          {link}{" "}
          <span className="arrow" aria-hidden="true">
            &#8599;
          </span>
        </Link>
      </div>
    </section>
  );
}
