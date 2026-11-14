import { test } from "node:test";
import { strict as assert } from "node:assert";
import { resolveApiBaseURL } from "./api-url.ts";

test("missing API configuration is rejected in every environment", () => {
  assert.equal(resolveApiBaseURL(undefined, "development"), "");
  assert.equal(resolveApiBaseURL(undefined, "production"), "");
});

test("production rejects loopback and malformed API URLs", () => {
  for (const url of ["http://localhost:8080/v1", "http://127.0.0.1:8080", "http://0.0.0.0:8080", "not a url"]) {
    assert.equal(resolveApiBaseURL(url, "production"), "");
  }
});

test("valid configured API URLs are normalized", () => {
  assert.equal(resolveApiBaseURL("https://api.example.test/v1///", "production"), "https://api.example.test/v1");
});
