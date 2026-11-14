"use client";

import Link from "next/link";
import { AlertCircle, ArrowLeft, CircleHelp, Home, RefreshCw, WifiOff } from "lucide-react";
import "./error-state.css";

export type ErrorStateKind = "offline" | "server" | "not_found" | "generic";

const content = {
  offline: {
    icon: WifiOff,
    eyebrow: "SEBENTAR YA",
    title: "Ruangmu belum bisa dibuka",
    description: "Coba periksa koneksi internetmu, lalu pilih coba lagi. Kami akan menyiapkan semuanya begitu kamu siap.",
  },
  server: {
    icon: AlertCircle,
    eyebrow: "SEDANG ADA GANGGUAN",
    title: "Relay sedang beristirahat sebentar",
    description: "Pesan dan ruangmu tetap aman. Coba lagi dalam beberapa saat.",
  },
  not_found: {
    icon: CircleHelp,
    eyebrow: "HALAMAN TIDAK DITEMUKAN",
    title: "Sepertinya tautan ini sudah tidak aktif",
    description: "Periksa kembali alamatnya, atau kembali ke ruang Relay.",
  },
  generic: {
    icon: AlertCircle,
    eyebrow: "TERJADI KESALAHAN",
    title: "Halaman ini belum bisa dibuka",
    description: "Coba muat ulang. Jika masalah berlanjut, kembali lagi sebentar nanti.",
  },
} satisfies Record<ErrorStateKind, { icon: typeof AlertCircle; eyebrow: string; title: string; description: string }>;

export function ErrorState({
  kind = "generic",
  onRetry,
  retryLabel = "Coba lagi",
  mainId,
}: {
  kind?: ErrorStateKind;
  onRetry?: () => void;
  retryLabel?: string;
  mainId?: string;
}) {
  const view = content[kind];
  const Icon = view.icon;

  return (
    <main className="system-state" id={mainId} aria-labelledby="system-state-title">
      <div className="system-state-card">
        <Link className="system-state-brand" href="/" aria-label="XYNOSS Relay, beranda">
          <span className="system-state-brand-mark" aria-hidden="true"><i /><i /><i /><i /></span>
          <span>XYNOSS <b>Relay</b></span>
        </Link>
        <div className={`system-state-icon system-state-icon-${kind}`} aria-hidden="true"><Icon size={25} strokeWidth={1.8} /></div>
        <p className="system-state-eyebrow">{view.eyebrow}</p>
        <h1 id="system-state-title">{view.title}</h1>
        <p className="system-state-description">{view.description}</p>
        <div className="system-state-actions">
          {onRetry && <button className="system-state-primary" onClick={onRetry}><RefreshCw size={16} />{retryLabel}</button>}
          <Link className={onRetry ? "system-state-secondary" : "system-state-primary"} href="/"><Home size={16} />Ke beranda</Link>
        </div>
        <Link className="system-state-back" href="/" onClick={(event) => { if (window.history.length > 1) { event.preventDefault(); window.history.back(); } }}><ArrowLeft size={14} /> Kembali ke halaman sebelumnya</Link>
      </div>
      <p className="system-state-footnote">Ruang percakapanmu tetap milikmu.</p>
    </main>
  );
}

export function LoadingState() {
  return (
    <main className="system-state" id="main-content" aria-live="polite" aria-busy="true">
      <div className="system-state-card system-state-loading">
        <span className="system-state-brand-mark" aria-hidden="true"><i /><i /><i /><i /></span>
        <span className="system-state-spinner" aria-hidden="true" />
        <h1>Menyiapkan ruangmu</h1>
        <p className="system-state-description">Sebentar, kami sedang menghubungkanmu kembali.</p>
      </div>
    </main>
  );
}
