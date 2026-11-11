import { test } from "node:test";
import { strict as assert } from "node:assert";
import { safeAuthRedirect } from "./safe-redirect.ts";

test("tujuan undangan dan aplikasi tetap tersedia", () => {
  for (const value of ["/app", "/app?join=abc", "/app/workspace/123", "/invite/user/teman"]) assert.equal(safeAuthRedirect(value), value);
});

test("tujuan eksternal, encoded slash, dan path lain ditolak", () => {
  for (const value of ["https://example.com", "//example.com", "/\\example.com", "javascript:alert(1)", "/app/%2f%2fexample.com", "/app/%5cexample.com", "/login", "/app/../login", "/app\n/evil", ""]) assert.equal(safeAuthRedirect(value), "/app", value);
});
