"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import Messenger from "./messenger";

export default function Page() {
  const router = useRouter();
  const [status, setStatus] = useState<"checking" | "ready" | "unavailable">("checking");
  const [accountId, setAccountId] = useState("");
  useEffect(() => {
    let active = true;
    async function check() {
      try {
        await api.auth.getSession();
        const account = await api.user.getMe();
        if (!account.user?.id) throw new Error("Identitas akun tidak tersedia");
        if (!account.onboarding_completed) {
          const destination = window.location.pathname + window.location.search;
          sessionStorage.setItem("xynoss_redirect", destination);
          router.replace("/onboarding");
          return;
        }
        if (active) { setAccountId(account.user.id); setStatus("ready"); }
      } catch (error) {
        if (!active) return;
        const code = typeof error === "object" && error !== null && "status" in error ? error.status : undefined;
        if (code === 401 || code === 403) {
          const destination = window.location.pathname + window.location.search;
          router.replace(`/login?redirect=${encodeURIComponent(destination)}`);
        } else setStatus("unavailable");
      }
    }
    void check();
    return () => { active = false; };
  }, [router]);
  if (status === "ready") return <Messenger key={accountId} accountId={accountId} />;
  return <main className="app-gate"><div className="app-gate-mark">✳</div><p>{status === "checking" ? "Menyiapkan ruang percakapan…" : "Koneksi ke layanan belum tersedia."}</p>{status === "unavailable" && <button onClick={() => window.location.reload()}>Coba lagi</button>}</main>;
}
