import { test } from "node:test";
import { strict as assert } from "node:assert";
import { closeSessions } from "./sign-out-core.ts";

test("logout menutup kedua sesi dan membersihkan cache", async () => {
  const calls = [];
  await closeSessions({ logoutApi: async () => calls.push("api"), getManagedSession: async () => ({data:{user:{id:"a"}}}), logoutManaged: async () => { calls.push("managed"); return {}; }, clearLocal: () => calls.push("clear") });
  assert.deepEqual(calls, ["api", "managed", "clear"]);
});

test("logout melaporkan kegagalan tetapi tetap mencoba menutup sesi lain", async () => {
  const calls = [];
  await assert.rejects(closeSessions({ logoutApi: async () => { calls.push("api"); throw new Error("offline"); }, getManagedSession: async () => ({data:{user:{id:"a"}}}), logoutManaged: async () => { calls.push("managed"); return {}; }, clearLocal: () => calls.push("clear") }), /belum dapat ditutup/);
  assert.deepEqual(calls, ["api", "managed", "clear"]);
});

test("tidak memanggil managed logout jika tidak ada sesi managed", async () => {
  let called = false;
  await closeSessions({ logoutApi: async () => {}, getManagedSession: async () => ({data:null}), logoutManaged: async () => { called = true; return {}; }, clearLocal: () => {} });
  assert.equal(called, false);
});
