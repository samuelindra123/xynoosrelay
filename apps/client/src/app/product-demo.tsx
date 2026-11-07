"use client";

import { useEffect, useState, useSyncExternalStore } from "react";

const devices = ["Ponsel", "Desktop", "Website"] as const;
type Device = (typeof devices)[number];
const motionQuery = "(prefers-reduced-motion: reduce)";
function subscribeMotion(callback: () => void) { const query = window.matchMedia(motionQuery); query.addEventListener("change", callback); return () => query.removeEventListener("change", callback); }
function getReducedMotion() { return window.matchMedia(motionQuery).matches; }
function getServerMotion() { return false; }

export function ProductDemo() {
  const [device, setDevice] = useState<Device>("Ponsel");
  const [playing, setPlaying] = useState(true);
  const reducedMotion = useSyncExternalStore(subscribeMotion, getReducedMotion, getServerMotion);

  useEffect(() => {
    if (!playing || reducedMotion) return;
    const timer = window.setInterval(() => {
      setDevice((current) => devices[(devices.indexOf(current) + 1) % devices.length]);
    }, 7000);
    return () => window.clearInterval(timer);
  }, [playing, reducedMotion]);

  function chooseDevice(next: Device) {
    setDevice(next);
    setPlaying(false);
  }

  return (
    <div className={`product-demo ${playing && !reducedMotion ? "is-playing" : "is-paused"}`} aria-label="Demo animasi konsep XYNOSS Relay">
      <div className="demo-stage">
        <div className="demo-halo halo-outer" /><div className="demo-halo halo-inner" />
        <div className="demo-floating demo-floating-top"><span>&#10035;</span><p>Obrolan kecil.<br /><strong>Rasa yang besar.</strong></p></div>
        <div className={`demo-device device-${device.toLowerCase()}`}>
          <div className="demo-browser"><span className="browser-dots"><i /><i /><i /></span><span className="browser-address">relay <span>/</span> percakapan</span><span className="browser-end">&#8599;</span></div>
          <div className="demo-phone-bar"><span>9.41</span><span>&#9679; &#9679; &#9644;</span></div>
          <div className="demo-app">
            <aside className="demo-sidebar" aria-hidden="true"><div className="demo-sidebar-title"><span className="demo-mini-mark">&#10035;</span> Relay <span>+</span></div><div className="demo-search">Cari percakapan...</div><div className="demo-contact active-contact"><span className="demo-avatar">N</span><div><b>Nara</b><small>Aku dengerin.</small></div></div><div className="demo-contact"><span className="demo-avatar avatar-alt">A</span><div><b>Awan</b><small>Sampai nanti!</small></div></div><div className="demo-contact"><span className="demo-avatar avatar-third">S</span><div><b>Sasa</b><small>Terima kasih ya</small></div></div><div className="demo-sidebar-bottom">Ruang untuk cerita &#10035;</div></aside>
            <div className="demo-chat"><div className="demo-chat-head"><span className="demo-back">&#8249;</span><span className="demo-avatar">N</span><div><strong>Nara</strong><small>teman ngobrol</small></div><span className="demo-more">&#8226;&#8226;&#8226;</span></div><div className="demo-chat-body"><div className="demo-date">HARI INI</div><div className="demo-bubble demo-in demo-msg-one">Ada waktu buat cerita<br />sebentar? <span>&#10035;</span></div><div className="demo-bubble demo-out demo-msg-two">Selalu ada buat kamu.</div><div className="demo-bubble demo-in demo-msg-three">Nah, jadi gini...</div><div className="demo-typing"><i /><i /><i /></div></div><div className="demo-input"><span>Pesan...</span><span className="demo-send">&#8599;</span></div></div>
          </div>
        </div>
        <div className="demo-floating demo-floating-bottom"><span>&#9786;</span><p>Satu cerita,<br /><strong>banyak cara dekat.</strong></p></div>
      </div>
      <div className="demo-controls"><div className="demo-label"><span className="demo-live-dot" /> DEMO KONSEP <span className="demo-counter">/ 0{devices.indexOf(device) + 1}</span></div><div className="demo-options" aria-label="Pilih tampilan demo">{devices.map((item) => <button key={item} type="button" className={device === item ? "selected" : ""} aria-pressed={device === item} onClick={() => chooseDevice(item)}>{item}</button>)}</div><button className="demo-play" type="button" aria-label={reducedMotion ? "Animasi dikurangi sesuai pengaturan perangkat" : playing ? "Jeda pergantian tampilan" : "Putar pergantian tampilan"} disabled={reducedMotion} onClick={() => setPlaying(!playing)}>{playing && !reducedMotion ? "?" : "?"}</button></div>
      <div className="demo-progress" aria-hidden="true"><span key={`${device}-${playing}`} /></div><p className="demo-disclaimer">ANIMASI ILUSTRATIF, BUKAN TAMPILAN ATAU KETERSEDIAAN FINAL</p>
    </div>
  );
}
