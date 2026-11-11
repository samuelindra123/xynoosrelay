"use client";

import { useEffect, useState, useMemo, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Check,
  CheckCircle2,
  ArrowRight,
  Loader2,
  AlertCircle,
  Link2Off,
  Compass,
} from "lucide-react";
import {
  api,
  ApiError,
  type InvitePreviewEntity,
} from "@/lib/api";
import { signOutEverywhere } from "@/lib/auth/sign-out";
import "./workspace-invite.css";

interface CurrentUser {
  id: string;
  name: string;
  username: string;
  avatarUrl?: string;
}

type CardState =
  | { type: "loading" }
  | { type: "invalid"; message?: string }
  | {
      type: "guest";
      preview: InvitePreviewEntity;
    }
  | {
      type: "member";
      preview: InvitePreviewEntity;
      user: CurrentUser;
    }
  | {
      type: "non_member";
      preview: InvitePreviewEntity;
      user: CurrentUser;
    };

interface WorkspacePalette {
  id: string;
  name: string;
  gradient: string;
  glow: string;
}

const WORKSPACE_PALETTES: WorkspacePalette[] = [
  {
    id: "violet",
    name: "Violet",
    gradient: "from-indigo-500 via-purple-600 to-pink-600",
    glow: "rgba(168, 85, 247, 0.22)",
  },
  {
    id: "cyan",
    name: "Cyan",
    gradient: "from-cyan-500 via-teal-600 to-blue-700",
    glow: "rgba(6, 182, 212, 0.22)",
  },
  {
    id: "emerald",
    name: "Emerald",
    gradient: "from-emerald-500 via-green-600 to-teal-800",
    glow: "rgba(16, 185, 129, 0.22)",
  },
  {
    id: "amber",
    name: "Amber",
    gradient: "from-amber-500 via-orange-600 to-rose-700",
    glow: "rgba(245, 158, 11, 0.22)",
  },
  {
    id: "blue",
    name: "Blue",
    gradient: "from-sky-500 via-blue-600 to-indigo-800",
    glow: "rgba(59, 130, 246, 0.22)",
  },
  {
    id: "rose",
    name: "Rose",
    gradient: "from-rose-500 via-pink-600 to-purple-800",
    glow: "rgba(244, 63, 94, 0.22)",
  },
];

function getPalette(seed: string): WorkspacePalette {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = seed.charCodeAt(i) + ((hash << 5) - hash);
  }
  return WORKSPACE_PALETTES[Math.abs(hash) % WORKSPACE_PALETTES.length];
}

function getMonogram(name: string): string {
  const clean = (name || "Workspace").trim();
  const words = clean.split(/\s+/).filter(Boolean);
  if (words.length >= 2) {
    return `${words[0][0]}${words[1][0]}`.toUpperCase();
  }
  return clean.slice(0, 2).toUpperCase();
}

export default function InviteCard({ code }: { code: string }) {
  const router = useRouter();
  const [state, setState] = useState<CardState>({ type: "loading" });
  const [isAccepting, setIsAccepting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [acceptError, setAcceptError] = useState<string | null>(null);

  // Initialize and evaluate 5-state machine
  useEffect(() => {
    let active = true;

    async function evaluateInvite() {
      try {
        // Step 1: Fetch public preview
        let preview: InvitePreviewEntity;
        try {
          preview = await api.invites.getPreview(code);
        } catch (err: unknown) {
          if (!active) return;
          const msg =
            err instanceof ApiError && err.status === 404
              ? "Tautan undangan tidak ditemukan atau sudah kedaluwarsa."
              : err instanceof Error ? err.message : "Tautan undangan tidak valid atau tidak dapat diakses.";
          setState({ type: "invalid", message: msg });
          return;
        }

        if (!active) return;

        // Step 2: Determine current user session
        let currentUser: CurrentUser | null = null;
        try {
          await api.auth.getSession();
          const meData = await api.user.getMe();
          if (meData && meData.user) {
            currentUser = {
              id: meData.user.id,
              name: meData.profile?.display_name || meData.user.display_name || "Pengguna",
              username: meData.profile?.username || meData.user.username || "pengguna",
              avatarUrl: meData.profile?.avatar_url || undefined,
            };
          }
        } catch {
          currentUser = null;
        }

        if (!active) return;

        // Step 3: Branch between Unauthenticated Guest (State C) and Authenticated (State D or E)
        if (!currentUser) {
          setState({ type: "guest", preview });
          return;
        }

        // Step 4: Check if already a member of this workspace
        let isMember = false;
        try {
          const userWorkspaces = await api.workspaces.list();
          if (Array.isArray(userWorkspaces)) {
            isMember = userWorkspaces.some((w) => w.id === preview.workspace_id);
          }
        } catch {
          isMember = false;
        }

        if (!active) return;

        if (isMember) {
          setState({ type: "member", preview, user: currentUser });
        } else {
          setState({ type: "non_member", preview, user: currentUser });
        }
      } catch (fatal: unknown) {
        if (!active) return;
        setState({
          type: "invalid",
          message: fatal instanceof Error ? fatal.message : "Terjadi kesalahan saat memeriksa undangan.",
        });
      }
    }

    evaluateInvite();

    return () => {
      active = false;
    };
  }, [code]);

  // Handle invitation acceptance (State E)
  const handleAccept = useCallback(async () => {
    if (state.type !== "non_member" || isAccepting || isSuccess) return;

    setIsAccepting(true);
    setAcceptError(null);

    try {
      const result = await api.invites.accept(code);
      setIsSuccess(true);
      const targetWorkspaceId = result?.id || state.preview.workspace_id;

      // Smooth transition to workspace
      setTimeout(() => {
        router.push(`/app?workspace=${encodeURIComponent(targetWorkspaceId)}`);
      }, 500);
    } catch (err: unknown) {
      setIsAccepting(false);
      setAcceptError(
        err instanceof Error ? err.message : "Gagal bergabung ke ruang kerja. Silakan coba kembali."
      );
    }
  }, [code, state, isAccepting, isSuccess, router]);

  // Deterministic palette and ambient glow computation
  const activeWorkspaceName = useMemo(() => {
    if (state.type === "guest" || state.type === "member" || state.type === "non_member") {
      return state.preview.workspace_name;
    }
    return code;
  }, [state, code]);

  const palette = useMemo(() => {
    return getPalette(activeWorkspaceName);
  }, [activeWorkspaceName]);

  const ambientGlow = useMemo(() => {
    if (state.type === "invalid") {
      return "rgba(239, 68, 68, 0.16)";
    }
    return palette.glow;
  }, [state.type, palette.glow]);

  return (
    <main className="workspace-invite-page">
      <Link href="/" className="workspace-invite-brand"><span>✳</span> relay.</Link>
      {/* Ambient background glows */}
      <div
        className="pointer-events-none absolute -top-44 -left-44 w-[640px] h-[640px] rounded-full blur-[150px] transition-colors duration-1000"
        style={{ background: ambientGlow }}
      />
      <div
        className="pointer-events-none absolute -bottom-44 -right-44 w-[640px] h-[640px] rounded-full blur-[150px] transition-colors duration-1000"
        style={{ background: ambientGlow }}
      />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.02)_0%,transparent_75%)]" />

      {/* Center Discord-style Invite Card */}
      <div className="workspace-invite-card relative w-full max-w-[460px] min-h-[480px] rounded-[28px] bg-[#20352a] border border-[#5a7560]/40 p-7 sm:p-9 flex flex-col justify-between shadow-2xl transition-all duration-300">
        {/* Subtle rim-glow on the card itself */}
        <div
          className="pointer-events-none absolute inset-0 rounded-2xl opacity-20 blur-xl transition-all duration-700"
          style={{ background: ambientGlow }}
        />

        {/* =========================================================================
            STATE A: SKELETON LOADING
           ========================================================================= */}
        {state.type === "loading" && (
          <>
            <div className="flex flex-col items-center text-center flex-1 justify-center py-2 animate-pulse">
              {/* Avatar Skeleton */}
              <div className="relative mb-4">
                <div className="w-20 h-20 rounded-2xl bg-white/[0.07] border border-white/[0.08]" />
              </div>

              {/* Header Meta Skeleton */}
              <div className="h-3.5 w-40 rounded-full bg-white/[0.06] mb-3" />

              {/* Title Skeleton */}
              <div className="h-7 w-52 rounded-lg bg-white/[0.08] mb-3" />

              {/* Description Skeleton */}
              <div className="h-4 w-64 rounded-full bg-white/[0.04] mb-5" />

              {/* Metrics Pill Skeleton */}
              <div className="h-8 w-44 rounded-full bg-white/[0.05] border border-white/[0.05]" />
            </div>

            <div className="mt-6 pt-2 animate-pulse flex flex-col gap-3">
              <div className="w-full h-11 rounded-xl bg-white/[0.08]" />
              <div className="h-3 w-36 rounded-full bg-white/[0.04] mx-auto" />
            </div>
          </>
        )}

        {/* =========================================================================
            STATE B: INVALID / EXPIRED CODE
           ========================================================================= */}
        {state.type === "invalid" && (
          <>
            <div className="flex flex-col items-center text-center flex-1 justify-center py-2">
              <div className="relative mb-5">
                <div className="absolute -inset-1 rounded-2xl bg-rose-500/20 blur-md" />
                <div className="relative w-16 h-16 rounded-2xl bg-rose-500/10 border border-rose-500/25 flex items-center justify-center text-rose-400 shadow-xl">
                  <Link2Off className="w-8 h-8" />
                </div>
              </div>

              <h2 className="text-lg sm:text-xl font-bold tracking-tight text-white px-2">
                Undangan Tidak Valid atau Sudah Kedaluwarsa
              </h2>

              <p className="text-xs sm:text-sm text-zinc-400 mt-2.5 max-w-[320px] leading-relaxed">
                {state.message ||
                  "Tautan undangan ini mungkin salah, telah kedaluwarsa, atau ditarik oleh pengundang."}
              </p>
            </div>

            <div className="mt-6 pt-2">
              <Link
                href="/"
                className="w-full h-11 rounded-xl bg-zinc-800/90 hover:bg-zinc-700/90 active:scale-[0.98] text-white font-medium text-sm transition-all duration-150 flex items-center justify-center gap-2 border border-white/[0.08] shadow-lg shadow-black/20"
              >
                <Compass className="w-4 h-4 text-zinc-400" />
                <span>Kembali ke Beranda</span>
              </Link>
            </div>
          </>
        )}

        {/* =========================================================================
            STATE C: UNAUTHENTICATED GUEST
           ========================================================================= */}
        {state.type === "guest" && (
          <>
            <div className="workspace-invite-overline"><span>UNDANGAN WORKSPACE</span><span>XYNOSS RELAY / 02</span></div>
            <div className="flex flex-col items-center text-center flex-1 justify-center py-2">
              {/* Workspace Avatar with ambient rim-glow */}
              <div className="relative mb-4">
                <div
                  className={`absolute -inset-1 rounded-2xl bg-gradient-to-tr ${palette.gradient} blur-md opacity-75`}
                />
                <div
                  className={`relative w-20 h-20 rounded-2xl bg-gradient-to-tr ${palette.gradient} flex items-center justify-center text-white font-bold text-2xl tracking-tight shadow-xl ring-1 ring-white/20`}
                >
                  {getMonogram(state.preview.workspace_name)}
                </div>
              </div>

              {/* Header Meta */}
              <div className="text-[11px] sm:text-xs font-medium uppercase tracking-wider text-zinc-400">
                {state.preview.inviter_name ? (
                  <span>
                    <strong className="text-zinc-300 font-semibold">
                      @{state.preview.inviter_name}
                    </strong>{" "}
                    mengundang Anda ke
                  </span>
                ) : (
                  <span>Diundang untuk bergabung dengan</span>
                )}
              </div>

              {/* Title */}
              <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-white mt-1 truncate max-w-full px-2">
                {state.preview.workspace_name}
              </h1>

              {/* Description */}
              {state.preview.workspace_description ? (
                <p className="text-xs sm:text-sm text-zinc-400 mt-2 line-clamp-2 max-w-[340px] leading-relaxed">
                  {state.preview.workspace_description}
                </p>
              ) : null}

              {/* Metrics Indicator */}
              <div className="flex items-center justify-center gap-4 py-1.5 px-3.5 rounded-full bg-white/[0.04] border border-white/[0.07] text-xs font-medium text-zinc-300 mt-4 mx-auto">
                <div className="flex items-center gap-1.5 text-zinc-400">
                  <span className="h-2 w-2 rounded-full bg-zinc-500" />
                  <span>{state.preview.member_count} Anggota</span>
                </div>
              </div>
            </div>

            {/* Bottom Actions */}
            <div className="mt-6 pt-2 flex flex-col gap-2.5">
              <button
                type="button"
                onClick={() => {
                  router.push(`/login?redirect=${encodeURIComponent(`/app?join=${code}`)}`);
                }}
                className="w-full h-11 rounded-xl bg-white hover:bg-zinc-200 active:scale-[0.98] text-zinc-950 font-semibold text-sm transition-all duration-150 flex items-center justify-center gap-2 shadow-lg shadow-white/10"
              >
                <span>Masuk untuk bergabung</span>
                <ArrowRight className="w-4 h-4" />
              </button>

              <div className="text-center text-xs text-zinc-400 mt-1">
                <span>Belum memiliki akun? </span>
                <Link
                  href={`/register?redirect=${encodeURIComponent(`/app?join=${code}`)}`}
                  onClick={() => {
                    if (typeof window !== "undefined") {
                      sessionStorage.setItem("xynoss_redirect", `/app?join=${code}`);
                    }
                  }}
                  className="text-zinc-200 hover:text-white underline underline-offset-4 font-medium transition-colors"
                >
                  Daftar Sekarang
                </Link>
              </div>
            </div>
          </>
        )}

        {/* =========================================================================
            STATE D: AUTHENTICATED MEMBER (ALREADY JOINED)
           ========================================================================= */}
        {state.type === "member" && (
          <>
            <div className="flex flex-col items-center text-center flex-1 justify-center py-2">
              {/* Workspace Avatar */}
              <div className="relative mb-4">
                <div
                  className={`absolute -inset-1 rounded-2xl bg-gradient-to-tr ${palette.gradient} blur-md opacity-75`}
                />
                <div
                  className={`relative w-20 h-20 rounded-2xl bg-gradient-to-tr ${palette.gradient} flex items-center justify-center text-white font-bold text-2xl tracking-tight shadow-xl ring-1 ring-white/20`}
                >
                  {getMonogram(state.preview.workspace_name)}
                </div>
              </div>

              <div className="text-[11px] sm:text-xs font-medium uppercase tracking-wider text-zinc-400">
                Ruang Kerja
              </div>

              {/* Title */}
              <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-white mt-1 truncate max-w-full px-2">
                {state.preview.workspace_name}
              </h1>

              {/* Description */}
              {state.preview.workspace_description ? (
                <p className="text-xs sm:text-sm text-zinc-400 mt-2 line-clamp-2 max-w-[340px] leading-relaxed">
                  {state.preview.workspace_description}
                </p>
              ) : null}

              {/* Metrics Indicator */}
              <div className="flex items-center justify-center gap-4 py-1.5 px-3.5 rounded-full bg-white/[0.04] border border-white/[0.07] text-xs font-medium text-zinc-300 mt-4 mx-auto">
                <div className="flex items-center gap-1.5 text-zinc-400">
                  <span className="h-2 w-2 rounded-full bg-zinc-500" />
                  <span>{state.preview.member_count} Anggota</span>
                </div>
              </div>
            </div>

            {/* Bottom Actions */}
            <div className="mt-6 pt-2 flex flex-col gap-3">
              {/* Already member banner */}
              <div className="flex items-center justify-center gap-2 py-2 px-3.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-xs font-medium">
                <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                <span>Anda sudah menjadi anggota ruang kerja ini</span>
              </div>

              <button
                type="button"
                onClick={() => {
                  router.push(`/app?workspace=${encodeURIComponent(state.preview.workspace_id)}`);
                }}
                className="w-full h-11 rounded-xl bg-white hover:bg-zinc-200 active:scale-[0.98] text-zinc-950 font-semibold text-sm transition-all duration-150 flex items-center justify-center gap-2 shadow-lg shadow-white/10"
              >
                <span>Lanjut ke Workspace</span>
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
          </>
        )}

        {/* =========================================================================
            STATE E: AUTHENTICATED NON-MEMBER (READY TO ACCEPT)
           ========================================================================= */}
        {state.type === "non_member" && (
          <>
            <div className="flex flex-col items-center text-center flex-1 justify-center py-2">
              {/* Workspace Avatar */}
              <div className="relative mb-4">
                <div
                  className={`absolute -inset-1 rounded-2xl bg-gradient-to-tr ${palette.gradient} blur-md opacity-75`}
                />
                <div
                  className={`relative w-20 h-20 rounded-2xl bg-gradient-to-tr ${palette.gradient} flex items-center justify-center text-white font-bold text-2xl tracking-tight shadow-xl ring-1 ring-white/20`}
                >
                  {getMonogram(state.preview.workspace_name)}
                </div>
              </div>

              {/* Header Meta */}
              <div className="text-[11px] sm:text-xs font-medium uppercase tracking-wider text-zinc-400">
                {state.preview.inviter_name ? (
                  <span>
                    <strong className="text-zinc-300 font-semibold">
                      @{state.preview.inviter_name}
                    </strong>{" "}
                    mengundang Anda ke
                  </span>
                ) : (
                  <span>Diundang untuk bergabung dengan</span>
                )}
              </div>

              {/* Title */}
              <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-white mt-1 truncate max-w-full px-2">
                {state.preview.workspace_name}
              </h1>

              {/* Description */}
              {state.preview.workspace_description ? (
                <p className="text-xs sm:text-sm text-zinc-400 mt-2 line-clamp-2 max-w-[340px] leading-relaxed">
                  {state.preview.workspace_description}
                </p>
              ) : null}

              {/* Metrics Indicator */}
              <div className="flex items-center justify-center gap-4 py-1.5 px-3.5 rounded-full bg-white/[0.04] border border-white/[0.07] text-xs font-medium text-zinc-300 mt-4 mx-auto">
                <div className="flex items-center gap-1.5 text-zinc-400">
                  <span className="h-2 w-2 rounded-full bg-zinc-500" />
                  <span>{state.preview.member_count} Anggota</span>
                </div>
              </div>
            </div>

            {/* Bottom Actions */}
            <div className="mt-6 pt-2 flex flex-col gap-3">
              {/* Current user chip */}
              <div className="flex items-center justify-between px-3 py-2 rounded-xl bg-white/[0.03] border border-white/[0.06] text-xs text-zinc-400">
                <div className="flex items-center gap-2 truncate">
                  <div className="w-5 h-5 rounded-full bg-zinc-800 border border-white/10 flex items-center justify-center text-[10px] text-zinc-200 font-bold shrink-0">
                    {state.user.username.charAt(0).toUpperCase()}
                  </div>
                  <span className="truncate">
                    Bergabung sebagai{" "}
                    <strong className="text-zinc-200 font-semibold">
                      @{state.user.username}
                    </strong>
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    void signOutEverywhere().then(() => window.location.reload()).catch(() => {
                      setAcceptError("Sesi belum dapat ditutup. Periksa koneksi lalu coba lagi.");
                    });
                  }}
                  className="text-[11px] text-zinc-400 hover:text-white transition-colors shrink-0 underline ml-2"
                >
                  Ganti Akun
                </button>
              </div>

              {/* Inline Destructive Error Alert */}
              {acceptError && (
                <div className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs text-left animate-in fade-in duration-200">
                  <AlertCircle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
                  <span className="leading-relaxed">{acceptError}</span>
                </div>
              )}

              {/* Primary Action Button */}
              <button
                type="button"
                disabled={isAccepting || isSuccess}
                onClick={handleAccept}
                className="w-full h-11 rounded-xl bg-white hover:bg-zinc-200 active:scale-[0.98] disabled:opacity-75 disabled:pointer-events-none text-zinc-950 font-semibold text-sm transition-all duration-150 flex items-center justify-center gap-2 shadow-lg shadow-white/10 relative overflow-hidden"
              >
                {isAccepting ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin text-zinc-900" />
                    <span>Memproses Undangan...</span>
                  </>
                ) : isSuccess ? (
                  <>
                    <Check className="w-4 h-4 text-emerald-600" />
                    <span className="text-emerald-700 font-bold">Berhasil Bergabung!</span>
                  </>
                ) : (
                  <>
                    <span>Terima Undangan</span>
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </div>
          </>
        )}
      </div>
    </main>
  );
}
