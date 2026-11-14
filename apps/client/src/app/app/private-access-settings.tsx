"use client";

import { useEffect, useRef, useState } from "react";
import { api, ApiError, type WorkspaceDetailEntity } from "@/lib/api";

export default function PrivateAccessSettings({ workspace, channelId }: { workspace: WorkspaceDetailEntity; channelId: string }) {
  const [users, setUsers] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [attempt, setAttempt] = useState(0);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState("");
  const [blocked, setBlocked] = useState(false);
  const pending = useRef(false);
  useEffect(() => {
    let active = true;
    api.channelMessages.privateAccess(channelId).then(data => { if (active) { setUsers(data.user_ids); setBlocked(false); } })
      .catch(err => { if (active) { const denied = err instanceof ApiError && [401,403,404].includes(err.status); setBlocked(denied); setUsers([]); setError(denied ? "Pengaturan tidak tersedia atau akses kamu telah berubah." : err instanceof Error ? err.message : "Daftar akses belum dapat dimuat."); } })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [channelId, attempt]);

  async function change(userId: string, grant: boolean) {
    if (pending.current || blocked || (!grant && !window.confirm("Cabut akses anggota ini ke channel privat?"))) return;
    pending.current = true; setSaving(userId); setError(""); setNotice("");
    try {
      if (grant) await api.channelMessages.grantPrivateAccess(channelId, userId);
      else await api.channelMessages.revokePrivateAccess(channelId, userId);
      setUsers(previous => grant ? [...new Set([...previous, userId])] : previous.filter(id => id !== userId));
      setNotice(grant ? "Akses berhasil diberikan." : "Akses berhasil dicabut.");
    } catch (err) {
      const denied = err instanceof ApiError && [401,403,404].includes(err.status);
      if (denied) { setBlocked(true); setUsers([]); }
      setError(denied ? "Pengaturan tidak tersedia atau akses kamu telah berubah." : err instanceof Error ? err.message : "Perubahan akses gagal. Coba lagi.");
    } finally { pending.current = false; setSaving(""); }
  }

  const others = (workspace.members || []).filter(member => member.user_id !== workspace.owner_user_id);
  const unknownIds = users.filter(id => id !== workspace.owner_user_id && !others.some(member => member.user_id === id));
  return <section aria-label="Pengaturan akses channel privat" aria-busy={loading}>
    <span className="modal-header-tag">AKSES PRIVAT</span><h2>Akses channel privat</h2>
    <p>Owner selalu memiliki akses. Anggota lain membutuhkan izin di bawah ini.</p>
    <button className="secondary-button" disabled={loading || !!saving} onClick={() => { setLoading(true); setError(""); setAttempt(value => value + 1); }}>Muat ulang akses</button>
    {loading && <p role="status">Memuat daftar akses…</p>}
    {error && <p role="alert">{error}</p>}
    {!loading && !blocked && <div className="access-members">
      <p><b>Owner:</b> {workspace.members?.find(member => member.user_id === workspace.owner_user_id)?.display_name || "Owner workspace"} · Akses tetap</p>
      {!users.length && <p>Belum ada izin tambahan. Hanya owner yang dapat mengakses.</p>}
      {others.map(member => <div className="session-row" key={member.user_id}>
        <span><b>{member.display_name}</b><small>@{member.username} · {member.role}</small><small>{users.includes(member.user_id) ? "Memiliki akses" : "Belum memiliki akses"}</small></span>
        <button disabled={!!saving} onClick={() => void change(member.user_id, !users.includes(member.user_id))}>{saving === member.user_id ? "Menyimpan…" : users.includes(member.user_id) ? "Cabut akses" : "Beri akses"}</button>
      </div>)}
      {unknownIds.map(id => <div className="session-row" key={id}><span>Izin pengguna di luar daftar anggota<small>{id}</small></span><button disabled={!!saving} onClick={() => void change(id, false)}>Cabut akses</button></div>)}
      {!others.length && <p>Belum ada anggota lain. Undang anggota sebelum memberikan akses.</p>}
    </div>}
    {notice && <p role="status">{notice}</p>}
  </section>;
}
