import { test } from "node:test";
import { strict as assert } from "node:assert";
import { webcrypto } from "node:crypto";
import { pendingChannelRequest } from "./channel-retry.ts";

function storage() {
  const values = new Map();
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
    values,
  };
}

test("kunci tetap sama setelah refresh dan berubah setelah kiriman berhasil", async () => {
  const store = storage();
  const first = await pendingChannelRequest(store, webcrypto, "akun-a", "channel-a", "pesan pribadi", 1000);
  const afterReload = await pendingChannelRequest(store, webcrypto, "akun-a", "channel-a", "pesan pribadi", 2000);
  assert.deepEqual(first, afterReload);
  assert.ok(!JSON.stringify([...store.values]).includes("pesan pribadi"));
  store.removeItem(first.storageKey);
  const newSend = await pendingChannelRequest(store, webcrypto, "akun-a", "channel-a", "pesan pribadi", 3000);
  assert.notEqual(first.key, newSend.key);
});

test("akun dan channel tidak berbagi kunci percobaan kirim", async () => {
  const store = storage();
  const a = await pendingChannelRequest(store, webcrypto, "akun-a", "channel-a", "halo", 1000);
  const b = await pendingChannelRequest(store, webcrypto, "akun-b", "channel-a", "halo", 1000);
  const c = await pendingChannelRequest(store, webcrypto, "akun-a", "channel-b", "halo", 1000);
  assert.notEqual(a.key, b.key);
  assert.notEqual(a.key, c.key);
});

test("percobaan lama atau penyimpanan rusak tidak diam-diam membuat kunci baru", async () => {
  const store = storage();
  const first = await pendingChannelRequest(store, webcrypto, "a", "c", "halo", 1000);
  await assert.rejects(pendingChannelRequest(store, webcrypto, "a", "c", "halo", 1000 + 23 * 60 * 60 * 1000), /terlalu lama/);
  store.setItem(first.storageKey, "bukan JSON");
  await assert.rejects(pendingChannelRequest(store, webcrypto, "a", "c", "halo", 2000), /tidak dapat dibaca/);
  const broken = { getItem() { return null; }, setItem() { throw new Error("quota"); } };
  await assert.rejects(pendingChannelRequest(broken, webcrypto, "a", "c", "halo", 1000), /quota/);
});
