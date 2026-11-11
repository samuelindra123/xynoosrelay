// A retry key is scoped to the account, channel and message fingerprint. Only
// the digest and random UUID are stored; the message body is not persisted here.
const MAX_AGE_MS = 23 * 60 * 60 * 1000;

export async function pendingChannelRequest(
  storage: Storage,
  cryptoProvider: Crypto,
  accountId: string,
  channelId: string,
  text: string,
  now = Date.now(),
): Promise<{ key: string; storageKey: string }> {
  const data = new TextEncoder().encode(JSON.stringify([accountId, channelId, text]));
  const digest = await cryptoProvider.subtle.digest("SHA-256", data);
  const fingerprint = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
  const storageKey = `xynoss_channel_retry_v1_${fingerprint}`;
  const existing = storage.getItem(storageKey);
  if (existing !== null) {
    let parsed: { key?: unknown; createdAt?: unknown };
    try { parsed = JSON.parse(existing); } catch { throw new Error("Data percobaan kirim tidak dapat dibaca."); }
    if (typeof parsed.key !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(parsed.key) || typeof parsed.createdAt !== "number" || !Number.isFinite(parsed.createdAt)) {
      throw new Error("Data percobaan kirim tidak valid.");
    }
    if (now < parsed.createdAt || now - parsed.createdAt >= MAX_AGE_MS) {
      throw new Error("Percobaan kirim terlalu lama untuk diulang dengan aman. Periksa riwayat pesan.");
    }
    return { key: parsed.key, storageKey };
  }
  const key = cryptoProvider.randomUUID();
  storage.setItem(storageKey, JSON.stringify({ key, createdAt: now }));
  return { key, storageKey };
}
