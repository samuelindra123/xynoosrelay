import { test } from "node:test";
import { strict as assert } from "node:assert";
import { validateProfileFile } from "./media-validation.ts";

test("client media limits match Go: avatar 5 MiB and banner 10 MiB", () => {
  for (const type of ["image/jpeg", "image/png", "image/webp"]) {
    assert.equal(validateProfileFile({ type, size: 5 * 1024 * 1024 }, "avatar"), null);
    assert.equal(validateProfileFile({ type, size: 10 * 1024 * 1024 }, "banner"), null);
  }
  assert.match(validateProfileFile({ type: "image/png", size: 5 * 1024 * 1024 + 1 }, "avatar"), /5 MB/);
  assert.match(validateProfileFile({ type: "image/png", size: 10 * 1024 * 1024 + 1 }, "banner"), /10 MB/);
  assert.match(validateProfileFile({ type: "image/png", size: 0 }, "avatar"), /lebih dari 0/);
  for (const type of ["image/svg+xml", "text/html", "image/gif", ""]) assert.match(validateProfileFile({ type, size: 100 }, "avatar"), /JPEG, PNG, atau WebP/);
});
