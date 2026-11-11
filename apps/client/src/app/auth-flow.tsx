"use client";

import { createContext, useContext, useEffect, useState, type FormEvent, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import Image from "next/image";
import { Header, Footer, Mark } from "./shared";
import { api } from "@/lib/api";
import { safeAuthRedirect } from "@/lib/safe-redirect";
import "./auth.css";

interface AuthContextType {
  email: string;
  setEmail: (value: string) => void;
  challengeId: string;
  setChallengeId: (value: string) => void;
}

const AuthContext = createContext<AuthContextType>({
  email: "",
  setEmail: () => {},
  challengeId: "",
  setChallengeId: () => {},
});

const authPaths = ["/login", "/register", "/verify-email", "/onboarding", "/app"];

export function AppShell({ children }: { children: ReactNode }) {
  const path = usePathname();
  const [email, setEmail] = useState("");
  const [challengeId, setChallengeId] = useState("");

  useEffect(() => {
    if (typeof window !== "undefined") {
      const storedEmail = sessionStorage.getItem("xynoss_email");
      const storedChallenge = sessionStorage.getItem("xynoss_challenge_id");
      // Restore a pending email challenge after browser hydration.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      if (storedEmail) setEmail(storedEmail);
      if (storedChallenge) setChallengeId(storedChallenge);
    }
  }, []);

  const isStandalone = authPaths.includes(path) || path.startsWith("/invite");

  return (
    <AuthContext.Provider value={{ email, setEmail, challengeId, setChallengeId }}>
      {isStandalone ? (
        children
      ) : (
        <>
          <Header />
          <div id="main-content">{children}</div>
          <Footer />
        </>
      )}
    </AuthContext.Provider>
  );
}

export function AuthFlow() {
  const path = usePathname();
  const router = useRouter();
  const { email, setEmail, challengeId, setChallengeId } = useContext(AuthContext);

  const [input, setInput] = useState("");
  const [otp, setOtp] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [cooldown, setCooldown] = useState(0);
  const [done, setDone] = useState(false);

  const verify = path === "/verify-email";
  const onboarding = path === "/onboarding";
  const register = path === "/register";

  useEffect(() => {
    if (!verify || cooldown <= 0) return;
    const timer = window.setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [verify, cooldown]);

  // Sync state from sessionStorage if context was lost on refresh
  useEffect(() => {
    if (typeof window !== "undefined") {
      if (!email) {
        const stored = sessionStorage.getItem("xynoss_email");
        if (stored) setEmail(stored);
      }
      if (!challengeId) {
        const storedC = sessionStorage.getItem("xynoss_challenge_id");
        if (storedC) setChallengeId(storedC);
      }
    }
  }, [email, challengeId, setEmail, setChallengeId]);

  useEffect(() => {
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search);
      const redirectParam = params.get("redirect");
      if (redirectParam) {
        sessionStorage.setItem("xynoss_redirect", safeAuthRedirect(redirectParam));
      }
      const oauthErr = params.get("error");
      if (oauthErr) {
        // OAuth returns the failure through the browser's redirect URL.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setError(oauthErr === "account_not_linked" ? "Email ini sudah digunakan. Masuk dengan metode sebelumnya untuk mengelola akun." : oauthErr === "access_denied" ? "Kamu membatalkan proses masuk dengan Google." : "Belum bisa masuk dengan Google. Coba lagi atau gunakan email.");
      }
    }
  }, []);

  async function submitEmail(e: FormEvent) {
    e.preventDefault();
    const cleanEmail = input.trim().toLowerCase();
    if (cleanEmail.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(cleanEmail)) {
      setError("Masukkan alamat email yang valid."); return;
    }
    setError(""); setBusy(true);
    try {
      const result = await api.auth.requestOtp(cleanEmail);
      if (!result.challenge_id || result.challenge_id === "cooldown") throw new Error("cooldown");
      setEmail(cleanEmail);
      sessionStorage.setItem("xynoss_email", cleanEmail);
      sessionStorage.removeItem("xynoss_dev_otp");
      sessionStorage.setItem("xynoss_challenge_id", result.challenge_id);
      setChallengeId(result.challenge_id);
      setOtp(""); setCooldown(60);
      router.push("/verify-email");
    } catch {
      setError("Kode belum dapat dikirim. Periksa alamat email dan coba lagi.");
    } finally { setBusy(false); }
  }

  async function submitOtp(e: FormEvent) {
    e.preventDefault();
    const cleanOtp = otp.trim();
    const activeEmail = email || sessionStorage.getItem("xynoss_email") || "";
    if (!/^\d{6}$/.test(cleanOtp)) { setError("Masukkan 6 digit kode verifikasi."); return; }
    if (!activeEmail) { setError("Masukkan alamat email kembali untuk meminta kode."); return; }
    setError(""); setBusy(true);
    try {
      const challenge = challengeId || sessionStorage.getItem("xynoss_challenge_id");
      if (!challenge) { setError("Minta kode baru untuk melanjutkan."); return; }
      const result = await api.auth.verifyOtp(challenge, cleanOtp);
      sessionStorage.removeItem("xynoss_email");
      sessionStorage.removeItem("xynoss_challenge_id");
      setOtp("");
      setDone(true);
      if (result.onboarding_required) router.push("/onboarding");
      else {
        const destination = safeAuthRedirect(sessionStorage.getItem("xynoss_redirect"));
        sessionStorage.removeItem("xynoss_redirect");
        router.push(destination);
      }
    } catch {
      setError("Belum bisa menyelesaikan proses masuk. Periksa koneksi lalu coba lagi.");
    } finally { setBusy(false); }
  }

  async function resendOtp() {
    if (busy || cooldown > 0) return;
    const activeEmail = email || sessionStorage.getItem("xynoss_email") || "";
    if (!activeEmail) { setError("Masukkan alamat email kembali untuk meminta kode."); return; }
    setError(""); setNotice(""); setBusy(true);
    try {
      const result = await api.auth.requestOtp(activeEmail);
      if (!result.challenge_id || result.challenge_id === "cooldown") throw new Error("cooldown");
      setChallengeId(result.challenge_id);
      sessionStorage.setItem("xynoss_challenge_id", result.challenge_id);
      setOtp(""); setCooldown(60);
      setNotice("Kode baru telah diminta. Periksa kotak masuk emailmu.");
    } catch { setError("Kode belum dapat dikirim ulang. Tunggu sebentar lalu coba lagi."); }
    finally { setBusy(false); }
  }
  async function submitName(e: FormEvent) {
    e.preventDefault();
    const cleanName = name.trim();
    if (cleanName.length < 2) { setError("Nama tampilan minimal 2 karakter."); return; }
    setError(""); setBusy(true);
    try {
      await api.user.completeOnboarding({ display_name: cleanName });
      setDone(true);
      const destination = safeAuthRedirect(sessionStorage.getItem("xynoss_redirect"));
      sessionStorage.removeItem("xynoss_redirect");
      router.push(destination);
    } catch { setError("Profil belum dapat disimpan. Periksa koneksi dan coba lagi."); }
    finally { setBusy(false); }
  }

  async function google() {
    setError(""); setBusy(true);
    try {
      const destination = safeAuthRedirect(sessionStorage.getItem("xynoss_redirect"));
      window.location.assign(api.auth.getGoogleStartUrl(destination));
    } catch {
      setError("Belum bisa masuk dengan Google. Coba lagi atau gunakan email.");
      setBusy(false);
    }
  }

  return (
    <>
      <main className="auth-screen" id="main-content">
        <div className="auth-top">
          <Link className="brand" href="/">
            <Mark />
            <span>
              XYNOSS <strong>Relay</strong>
            </span>
          </Link>
          <Link className="auth-exit" href="/">
            Tutup <span aria-hidden="true">&#10005;</span>
          </Link>
        </div>
        <div className="auth-grid">
          <section className="auth-stage">
            <div className="auth-stage-top">
              <span>RUANG UNTUK CERITA</span>
              <span>01 / 03</span>
            </div>
            <div className="auth-orbit" aria-hidden="true">
              <div className="auth-orbit-line" />
              <div className="auth-orbit-line two" />
              <div className="auth-orb">
                <span>&#10035;</span>
              </div>
              <div className="auth-note note-one">
                Hei, kamu di sini <span>&#9825;</span>
              </div>
              <div className="auth-note note-two">Cerita baru dimulai &#8599;</div>
            </div>
            <div className="auth-stage-copy">
              <p>LEBIH DEKAT, SEJAK AWAL.</p>
              <h2>
                Satu sapaan.
                <br />
                <em>Banyak cerita.</em>
              </h2>
              <span>Messaging without the weight.</span>
            </div>
          </section>
          <section className="auth-main">
            <div className="auth-inner">
              <div className="auth-meta">
                <span>
                  <i /> XYNOSS RELAY
                </span>
                <span>{onboarding ? "03" : verify ? "02" : "01"} / 03</span>
              </div>
              {onboarding ? (
                <>
                  <p className="auth-kicker">SEDIKIT TENTANG KAMU</p>
                  <h1>
                    Senang
                    <br />
                    <em>berkenalan.</em>
                  </h1>
                  {done ? (
                    <div className="auth-success" role="status">
                      <b>Profil siap, {name.trim()}.</b>
                      <p>Selamat datang di ruang percakapan XYNOSS Relay.</p>
                      <Link href="/app">Buka messenger &#8599;</Link>
                    </div>
                  ) : (
                    <>
                      <p className="auth-desc">
                        Nama apa yang ingin kamu gunakan? Kamu bisa mulai dengan
                        nama panggilanmu.
                      </p>
                      <form onSubmit={submitName} noValidate>
                        <label htmlFor="profile-name">Nama tampilan</label>
                        <input
                          id="profile-name"
                          value={name}
                          onChange={(e) => setName(e.target.value)}
                          maxLength={50}
                          autoComplete="nickname"
                          placeholder="Misalnya, Nara"
                          aria-invalid={!!error}
                          aria-describedby={error ? "auth-error" : undefined}
                        />
                        {error && (
                          <p id="auth-error" className="auth-error" role="alert">
                            {error}
                          </p>
                        )}
                        <button className="auth-primary" disabled={busy}>
                          {busy ? "Menyimpan profil..." : "Selesaikan profil"}
                          <span>&#8599;</span>
                        </button>
                      </form>
                      <Link className="auth-back" href="/login">
                        &#8592; Batalkan
                      </Link>
                    </>
                  )}
                </>
              ) : verify ? (
                <>
                  <p className="auth-kicker">SATU LANGKAH LAGI</p>
                  <h1>
                    Cek kode
                    <br />
                    <em>kamu.</em>
                  </h1>
                  {done ? (
                    <div className="auth-success" role="status">
                      <b>Kode terverifikasi.</b>
                      <p>Menyiapkan sesi kamu...</p>
                      <Link className="auth-primary" href="/app">
                        Masuk ke messenger <span>&#8599;</span>
                      </Link>
                    </div>
                  ) : (
                    <>
                      <p className="auth-desc">
                        {email ? (
                          <>
                            Kode verifikasi 6 digit telah dikirimkan ke{" "}
                            <strong>{email}</strong>.
                          </>
                        ) : (
                          "Masukkan 6 digit kode yang dikirim ke email kamu."
                        )}
                      </p>


                      <form onSubmit={submitOtp} noValidate>
                        <label htmlFor="otp">Kode verifikasi 6 digit</label>
                        <input
                          className="auth-otp"
                          id="otp"
                          value={otp}
                          onChange={(e) =>
                            setOtp(
                              e.target.value.replace(/\D/g, "").slice(0, 6)
                            )
                          }
                          inputMode="numeric"
                          autoComplete="one-time-code"
                          maxLength={6}
                          placeholder="000000"
                          aria-invalid={!!error}
                          aria-describedby={error ? "auth-error" : undefined}
                        />
                        {error && (
                          <p id="auth-error" className="auth-error" role="alert">
                            {error}
                          </p>
                        )}
                        <button className="auth-primary" disabled={busy}>
                          {busy ? "Memeriksa..." : "Verifikasi kode"}
                          <span>&#8599;</span>
                        </button>
                      </form>
                      <div className="auth-resend">
                        <span>Belum menerima kode?</span>{" "}
                        <button disabled={cooldown > 0} onClick={resendOtp}>
                          {cooldown > 0
                            ? `Kirim ulang (${cooldown}s)`
                            : "Kirim ulang kode"}
                        </button>
                      </div>
                      {notice && (
                        <p className="auth-notice" role="status">
                          {notice}
                        </p>
                      )}
                      <Link className="auth-back" href="/login">
                        &#8592; Gunakan email lain
                      </Link>
                    </>
                  )}
                </>
              ) : (
                <>
                  <p className="auth-kicker">
                    {register ? "CERITA BARU DIMULAI" : "SELAMAT DATANG KEMBALI"}
                  </p>
                  <h1>
                    {register ? (
                      <>
                        Mari mulai
                        <br />
                        <em>dengan halo.</em>
                      </>
                    ) : (
                      <>
                        Senang kamu
                        <br />
                        <em>di sini.</em>
                      </>
                    )}
                  </h1>
                  {done ? (
                    <div className="auth-success" role="status">
                      <b>{notice}</b>
                      <Link className="auth-primary" href="/app">
                        Buka messenger <span>&#8599;</span>
                      </Link>
                    </div>
                  ) : (
                    <>
                      <p className="auth-desc">
                        Masuk atau daftar dalam satu langkah sederhana. Tanpa kata
                        sandi, tanpa perlu mengingat lagi.
                      </p>
                      <button
                        className="auth-google"
                        type="button"
                        onClick={google}
                        disabled={busy}
                      >
                        <Image
                          src="/google-g.png"
                          alt="Google"
                          width={20}
                          height={20}
                          className="google-g"
                        />
                        {busy ? "Menghubungkan..." : "Lanjutkan dengan Google"}
                      </button>
                      <div className="auth-divider">
                        <span />
                        atau gunakan email
                        <span />
                      </div>
                      <form onSubmit={submitEmail} noValidate>
                        <label htmlFor="email">Alamat email</label>
                        <input
                          id="email"
                          type="email"
                          value={input}
                          onChange={(e) => setInput(e.target.value)}
                          autoComplete="email"
                          placeholder="nama@email.com"
                          aria-invalid={!!error}
                          aria-describedby={error ? "auth-error" : undefined}
                        />
                        {error && (
                          <p id="auth-error" className="auth-error" role="alert">
                            {error}
                          </p>
                        )}
                        <button className="auth-primary" disabled={busy}>
                          {busy ? "Memproses..." : "Lanjutkan dengan email"}
                          <span>&#8599;</span>
                        </button>
                      </form>
                      <p className="auth-legal">
                        Dengan melanjutkan, kamu menyetujui{" "}
                        <Link href="/terms">ketentuan layanan</Link> dan{" "}
                        <Link href="/privacy">kebijakan privasi</Link> kami.
                      </p>
                    </>
                  )}
                </>
              )}
              <div className="auth-bottom">
                LESS APP. MORE CONVERSATION. <span>&#10035;</span>
              </div>
            </div>
          </section>
        </div>
      </main>
    </>
  );
}
