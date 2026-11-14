"use client";

import { useEffect, useRef, useState } from "react";
import { api, setCsrfToken, setSessionToken, type SessionEntity } from "@/lib/api";
import { useRouter } from "next/navigation";

export default function SessionSettings() {
  const router = useRouter();
  const [sessions, setSessions] = useState<SessionEntity[]>([]);
  const [loading, setLoading] = useState(true);
  const [attempt, setAttempt] = useState(0);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [revoking, setRevoking] = useState("");
  const pending = useRef(false);
  useEffect(() => {
    let active = true;
    api.user.getSessions().then(items => { if (active) setSessions(items); })
      .catch(err => { if (active) setError(err instanceof Error ? err.message : "Sesi belum dapat dimuat."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [attempt]);

  async function revoke(session: SessionEntity) {
    if (pending.current || !window.confirm(`Cabut sesi ${session.label || session.platform_hint || "perangkat ini"}?${session.is_current ? " Kamu akan keluar dari perangkat ini." : " Perangkat itu harus masuk kembali."}`)) return;
    pending.current = true; setRevoking(session.id); setError(""); setNotice("");
    try {
      await api.user.revokeSession(session.id);
      setSessions(items => items.filter(item => item.id !== session.id)); setNotice("Sesi berhasil dicabut.");
      if (session.is_current) { setCsrfToken(null); setSessionToken(null); router.replace("/login"); }
    } catch (err) { setError(err instanceof Error ? err.message : "Sesi gagal dicabut. Coba lagi."); }
    finally { pending.current = false; setRevoking(""); }
  }

  return <section aria-label="Sesi aktif" aria-busy={loading}>
    <p>Perangkat dan sesi aktif di akun kamu.</p>
    <button className="secondary-button" disabled={loading || !!revoking} onClick={() => { setLoading(true); setError(""); setAttempt(value => value + 1); }}>Muat ulang sesi</button>
    {loading && <p role="status">Memuat sesi…</p>}
    {error && <p role="alert">{error}</p>}
    {!loading && !error && !sessions.length && <p>Belum ada sesi aktif yang terdaftar.</p>}
    {!loading && sessions.map(session => <div className="session-row" key={session.id}>
      <span><b>{session.label || "Perangkat tanpa nama"}</b><small>{session.platform_hint || "Platform tidak tersedia"}{session.is_current ? " · Sesi saat ini" : ""}</small><small>Terakhir aktif: {new Date(session.last_seen_at).toLocaleString("id-ID")}</small><small>Berakhir: {new Date(Math.min(Date.parse(session.idle_expires_at), Date.parse(session.absolute_expires_at))).toLocaleString("id-ID")}</small></span>
      <button disabled={!!revoking} onClick={() => void revoke(session)}>{revoking === session.id ? "Mencabut…" : session.is_current ? "Cabut & keluar" : "Cabut sesi"}</button>
    </div>)}
    {notice && <p role="status">{notice}</p>}
  </section>;
}
