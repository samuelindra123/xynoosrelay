"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import Messenger from "./messenger";
import { ErrorState, LoadingState, type ErrorStateKind } from "../error-state";

export default function Page() {
  const router = useRouter();
  const [status, setStatus] = useState<"checking" | "ready" | "unavailable">("checking");
  const [accountId, setAccountId] = useState("");
  const [errorKind, setErrorKind] = useState<ErrorStateKind>("offline");
  const [attempt, setAttempt] = useState(0);
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
        } else {
          setErrorKind(error instanceof ApiError && error.kind === "offline" ? "offline" : error instanceof ApiError && error.kind === "server" ? "server" : error instanceof ApiError && error.kind === "not_found" ? "not_found" : "generic");
          setStatus("unavailable");
        }
      }
    }
    void check();
    return () => { active = false; };
  }, [router, attempt]);
  if (status === "ready") return <Messenger key={accountId} accountId={accountId} />;
  if (status === "checking") return <LoadingState />;
  return <ErrorState mainId="main-content" kind={errorKind} onRetry={() => { setStatus("checking"); setAttempt(value => value + 1); }} />;
}
