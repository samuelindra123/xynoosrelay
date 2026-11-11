type Dependencies = {
  logoutApi: () => Promise<unknown>;
  getManagedSession: () => Promise<{ data?: { user?: unknown } | null; error?: unknown }>;
  logoutManaged: () => Promise<{ error?: unknown }>;
  clearLocal: () => void;
};

export async function closeSessions(deps: Dependencies): Promise<void> {
  let failed = false;
  try { await deps.logoutApi(); } catch { failed = true; }
  try {
    const session = await deps.getManagedSession();
    if (session.error) failed = true;
    if (session.data?.user) {
      const result = await deps.logoutManaged();
      if (result.error) failed = true;
    }
  } catch { failed = true; }
  deps.clearLocal();
  if (failed) throw new Error("Sesi belum dapat ditutup sepenuhnya. Periksa koneksi lalu coba lagi.");
}
