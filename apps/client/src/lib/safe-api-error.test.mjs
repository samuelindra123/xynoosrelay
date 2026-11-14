import { test } from "node:test";
import { strict as assert } from "node:assert";
import { safeApiError } from "./safe-api-error.ts";

test("network and server failures return user-safe copy", () => {
  for (const status of [0, 500, 502, 503]) {
    const result = safeApiError(status);
    assert.ok(result.message.length > 0);
    assert.doesNotMatch(result.message, /localhost|http|sql|postgres|stack|exception|internal/i);
  }
});

test("common HTTP errors map to clear actions without backend payloads", () => {
  assert.equal(safeApiError(404).kind, "not_found");
  assert.equal(safeApiError(429).kind, "rate_limited");
  assert.equal(safeApiError(403).kind, "forbidden");
  assert.equal(safeApiError(400).kind, "request");
});
