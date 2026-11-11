import { api, setCsrfToken, setSessionToken } from "@/lib/api";

/** Close the application session and any active managed-auth browser session. */
export async function signOutEverywhere(): Promise<void> {
  try { await api.auth.logout(); }
  finally { setSessionToken(null); setCsrfToken(null); }
}
