import Link from "next/link";
import "./personal-invite.css";

export const instant = false;

export default async function PersonalInvite({ params }: { params: Promise<{ username: string }> }) {
  const { username } = await params;
  const name = username.replace(/^@/, "").slice(0, 32);
  const destination = `/app?chat_with=${encodeURIComponent(name)}`;
  return (
    <main className="personal-invite">
      <Link href="/" className="personal-brand" aria-label="XYNOSS Relay, beranda"><span>✳</span> relay.</Link>
      <section className="personal-card" aria-labelledby="invite-title">
        <div className="personal-card-top"><span>UNDANGAN PERSONAL</span><span>XYNOSS RELAY / 01</span></div>
        <div className="personal-orbit" aria-hidden="true"><span>✳</span><b>{name.slice(0, 2).toUpperCase() || "?"}</b><i>↗</i></div>
        <p className="personal-eyebrow">ADA YANG INGIN TERHUBUNG</p>
        <h1 id="invite-title">Obrolan baik<br />dimulai di sini<span>.</span></h1>
        <p className="personal-description">Kamu menerima tautan percakapan dari <strong>@{name}</strong>. Masuk untuk menemukan profilnya dan membuka percakapan.</p>
        <Link className="personal-action" href={destination}>Lanjutkan ke percakapan <span>↗</span></Link>
        <p className="personal-note">Akun diperlukan untuk melanjutkan. Ketersediaan pengguna akan diperiksa setelah masuk.</p>
      </section>
      <p className="personal-footer">Messaging without the weight.</p>
    </main>
  );
}
