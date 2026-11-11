import { test } from "node:test";
import { strict as assert } from "node:assert";
import { unreadCount, canMarkRead } from "./read-state.ts";
const messages = [{id: 1, author: "Nara"}, {id: 2, author: "Kamu"}, {id: 3, author: "Nara"}];
test("menghitung pesan masuk setelah penanda baca, tanpa pesan sendiri", () => {
  assert.equal(unreadCount(messages, 1), 1);
  assert.equal(unreadCount(messages, 3), 0);
  assert.equal(unreadCount(messages, undefined), 2);
  assert.equal(unreadCount(messages, 999), 0);
  assert.equal(unreadCount([], undefined), 0);
});
test("status baca memakai urutan server dan ID akun, bukan nama", () => {
  const data = [{id: 100, author: "Nama sama", authorId: "lain"}, {id: 101, author: "Nama sama", authorId: "saya"}, {id: 102, author: "Lain", authorId: "lain"}];
  assert.equal(unreadCount(data, 99, "saya"), 2);
  assert.equal(unreadCount(data, 100, "saya"), 1);
  assert.equal(unreadCount(data, 102, "saya"), 0);
});
test("hanya menandai baca saat chat terlihat dan berada di akhir", () => {
  const active = { visible: true, focused: true, covered: false, filtering: false, distanceFromBottom: 0 };
  assert.equal(canMarkRead(active), true);
  for (const change of [{visible:false}, {focused:false}, {covered:true}, {filtering:true}, {distanceFromBottom:25}]) assert.equal(canMarkRead({...active,...change}), false);
  assert.equal(canMarkRead({...active,distanceFromBottom:24}), true);
});
