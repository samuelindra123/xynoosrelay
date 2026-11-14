import { test } from "node:test";
import { strict as assert } from "node:assert";
import { readFile } from "node:fs/promises";
import ts from "typescript";

// Execute the actual API client with mocked HTTP, including its imported helpers.
let moduleId = 0;
async function loadApi(t) {
  const previous = process.env.NEXT_PUBLIC_API_URL;
  process.env.NEXT_PUBLIC_API_URL = "https://api.example.invalid/v1";
  t.after(() => { if (previous === undefined) delete process.env.NEXT_PUBLIC_API_URL; else process.env.NEXT_PUBLIC_API_URL = previous; });
  const compile = source => ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText;
  const url = source => `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`;
  let source = await readFile(new URL("./api.ts", import.meta.url), "utf8");
  for (const name of ["safe-api-error", "api-url"]) {
    const dependency = compile(await readFile(new URL(`./${name}.ts`, import.meta.url), "utf8"));
    source = source.replace(`"./${name}"`, JSON.stringify(url(dependency)));
  }
  return import(url(compile(source) + `\n// isolated module ${++moduleId}`));
}
const reply = (data, status = 200) => new Response(JSON.stringify({ data }), { status, headers: { "Content-Type": "application/json" } });

test("session list and profile media removal use the Go response shapes", async t => {
  const { api } = await loadApi(t);
  const calls = [];
  const session = { id: "session", label: "Firefox", platform_hint: "Linux", last_seen_at: "2026-11-14T01:00:00Z", is_current: true };
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, options]); return reply(url.endsWith("/sessions") ? [session] : { revision: 4, avatar_media_id: null }); });
  assert.deepEqual(await api.user.getSessions(), [session]);
  assert.equal((await api.user.deleteAvatar()).revision, 4);
  assert.equal((await api.user.deleteBanner()).revision, 4);
  assert.deepEqual(calls.slice(1).map(call => call[1].method), ["DELETE", "DELETE"]);
});

test("media intent, metadata, finalize and cleanup follow existing Go endpoints", async t => {
  const { api } = await loadApi(t);
  const calls = [];
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, options]); return options.method === "DELETE" ? new Response(null, { status: 204 }) : reply({ id: "media", status: "ready", url: "https://cdn.example.invalid/avatar.png" }); });
  await api.media.requestUpload({ purpose: "avatar", content_type: "image/png", size_bytes: 100 });
  assert.deepEqual(JSON.parse(calls[0][1].body), { purpose: "avatar", content_type: "image/png", size_bytes: 100 });
  assert.equal((await api.media.finalize("media")).status, "ready");
  await api.media.get("media"); await api.media.delete("media");
  assert.deepEqual(calls.map(call => new URL(call[0]).pathname), ["/v1/media/uploads", "/v1/media/media/finalize", "/v1/media/media", "/v1/media/media"]);
});

test("mutations keep cookie, in-memory Bearer, CSRF, revision and idempotency headers", async t => {
  const { api, setSessionToken, setCsrfToken } = await loadApi(t);
  setSessionToken("test-session"); setCsrfToken("test-csrf");
  const calls = [];
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, options]); return reply({ revision: 5, sequence: 1 }); });
  await api.user.updateProfile({ bio: "Hello" }, 4);
  assert.equal(calls[0][1].headers.get("If-Match"), "4");
  for (const client of [api.channelMessages, api.conversations]) {
    await client.send("target", "Hello", "same-retry-key");
    await client.send("target", "Hello", "same-retry-key");
  }
  for (const [, options] of calls) {
    assert.equal(options.credentials, "include");
    assert.equal(options.headers.get("Authorization"), "Bearer test-session");
    assert.equal(options.headers.get("X-CSRF-Token"), "test-csrf");
    assert.equal(options.headers.has("X-Internal-API-Key"), false);
  }
  assert.ok(calls.slice(1).every(([, options]) => options.headers.get("Idempotency-Key") === "same-retry-key"));
});

test("both message kinds support history, edit, delete, hearts and read cursors", async t => {
  const { api } = await loadApi(t);
  const calls = [];
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, options]); return reply({}); });
  for (const [client, history] of [[api.channelMessages, api.channelMessages.list], [api.conversations, api.conversations.messages]]) {
    await history("target", 101);
    await client.edit("target", 8, "Edited", 3);
    await client.delete("target", 8);
    await client.react("target", 8, true); await client.react("target", 8, false);
    await client.read("target"); await client.read("target", 8);
  }
  for (const offset of [0, 7]) {
    assert.ok(calls[offset][0].endsWith("?before_sequence=101"));
    assert.deepEqual(JSON.parse(calls[offset + 1][1].body), { text: "Edited", revision: 3 });
    assert.deepEqual(calls.slice(offset, offset + 7).map(([, options]) => options.method), ["GET", "PATCH", "DELETE", "PUT", "DELETE", "GET", "POST"]);
    assert.deepEqual(JSON.parse(calls[offset + 6][1].body), { last_read_sequence: 8 });
  }
});

test("workspace, private access and invitation management stay behind the same client auth", async t => {
  const { api } = await loadApi(t);
  const calls = [];
  t.mock.method(globalThis, "fetch", async (url, options) => { calls.push([url, options]); return reply({}); });
  await api.workspaces.get("workspace");
  await api.channelMessages.privateAccess("channel");
  await api.channelMessages.grantPrivateAccess("channel", "member");
  await api.channelMessages.revokePrivateAccess("channel", "member");
  await api.workspaces.createInvite("workspace", { duration_hours: 24, max_uses: 5 });
  await api.workspaces.listInvites("workspace"); await api.workspaces.revokeInvite("workspace", "invite");
  await api.invites.getPreview("code"); await api.invites.accept("code");
  assert.deepEqual(calls.map(([, options]) => options.method), ["GET", "GET", "PUT", "DELETE", "POST", "GET", "DELETE", "GET", "POST"]);
  assert.deepEqual(JSON.parse(calls[2][1].body), { user_id: "member" });
  assert.deepEqual(JSON.parse(calls[4][1].body), { duration_hours: 24, max_uses: 5 });
});

test("private errors and invalid successful responses never become silent empty data", async t => {
  const { api, ApiError } = await loadApi(t);
  t.mock.method(globalThis, "fetch", async () => new Response(JSON.stringify({ error: { code: "NOT_FOUND", message: "private workspace secret title" } }), { status: 404 }));
  await assert.rejects(api.channelMessages.privateAccess("hidden"), error => error instanceof ApiError && error.status === 404 && !error.message.includes("secret"));
  globalThis.fetch = async () => new Response("upstream HTML");
  await assert.rejects(api.user.getSessions(), error => error.code === "INVALID_RESPONSE");
  globalThis.fetch = async () => reply(undefined);
  await assert.rejects(api.user.getSessions(), error => error.code === "INVALID_RESPONSE");
});

test("signed upload honors method, headers and real progress without forwarding app credentials", async t => {
  const { api, setSessionToken, setCsrfToken } = await loadApi(t);
  setSessionToken("test-session"); setCsrfToken("test-csrf");
  const original = globalThis.XMLHttpRequest;
  t.after(() => { if (original === undefined) delete globalThis.XMLHttpRequest; else globalThis.XMLHttpRequest = original; });
  let request;
  globalThis.XMLHttpRequest = class {
    constructor() {
      // eslint-disable-next-line @typescript-eslint/no-this-alias -- Inspect the fake network request in this transport test.
      request = this;
      this.headers = {}; this.upload = {}; this.status = 200;
    }
    open(method, url) { this.method = method; this.url = url; }
    setRequestHeader(key, value) { this.headers[key] = value; }
    send() { this.upload.onprogress({ lengthComputable: true, loaded: 50, total: 100 }); this.onload(); }
  };
  const progress = [];
  const intent = { method: "PUT", upload_url: "https://storage.example.invalid/signed", expires_at: new Date(Date.now() + 60000).toISOString(), required_headers: { "Content-Type": "image/png", "x-amz-test": "signed" } };
  await api.media.uploadBinary(intent, new Blob(["image"]), percent => progress.push(percent));
  assert.equal(request.method, "PUT"); assert.equal(request.url, intent.upload_url);
  assert.deepEqual(request.headers, intent.required_headers); assert.deepEqual(progress, [50]);
  assert.notEqual(request.withCredentials, true);
  await assert.rejects(api.media.uploadBinary({ ...intent, expires_at: "2000-01-01T00:00:00Z" }, new Blob()), error => error.code === "UPLOAD_EXPIRED");
});
