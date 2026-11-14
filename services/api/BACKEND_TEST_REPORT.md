# Go Backend Test Report

Date: 2026-11-14 (updated)

## Scope and safety

This audit covered the Go module in `services/api`, including authentication,
OTP, email-provider selection, sessions, users, workspaces, channels,
messaging, media, storage, middleware, database helpers, routing, and health
handlers.

Database integration tests were run only against the expiring isolated Neon
branch `backend-audit-20261113142624`, created from the linked project and
configured through `TEST_DATABASE_URL` without writing credentials to the
repository. The production branch was never used as the test target. External
email was not sent; integration OTP used the in-memory email sandbox.

## Commands and verified results

| Command | Result |
| --- | --- |
| `go run ./cmd/migrate` against isolated branch | PASS; migrations 00001 and 00002 applied |
| `go run ./cmd/migrate` a second time | PASS; goose reported no migrations to run, version 2 |
| `go test ./... -count=1 -v` | PASS; 192 run events, 69 PASS events, 0 SKIP; 13 test-bearing packages passed |
| `go test ./... -count=1 -coverprofile=coverage.out` | PASS; server package 96.1%; toolchain emitted the profile as `services/api/coverage` |
| `go test -race ./... -count=1` | PASS with `CGO_ENABLED=1`, MSYS2 MinGW GCC 16.2.0 |
| `go vet ./...` | PASS |
| Live Resend/SMTP delivery | NOT RUN; sandbox and HTTP transport tests passed without contacting real recipients |
| `go test ./internal/server -run TestAPI_EndToEndRemainingAuthorizationMediaAndConcurrency -count=1 -v` | PASS against isolated branch; 1 test passed, 0 failed, 0 skipped |
| `go test ./... -count=1 -v` | PASS against isolated branch; all test-bearing packages passed, 0 failures, 0 skips |
| `go test -race ./... -count=1` | PASS with `CGO_ENABLED=1`, MSYS2 MinGW GCC |
| `go test ./... -count=1 -coverpkg=./... -coverprofile=coverage.out` | PASS; cross-package instrumentation completed, server coverage 92.1% |
| `go vet ./...` | PASS |
| `go test ./internal/server -run '^$' -count=1` | PASS; changed integration test compiled |

The final suite includes unit, handler, routing, storage-provider, media, HTTP
contract, and database-backed integration tests. A first isolated attempt
failed because the test harness inherited real email-provider settings from
`.env`; that harness defect was fixed by forcing the integration server to use
the in-memory sandbox, and the complete suite was rerun successfully.

## Follow-up verification

- The production-linked `.env` database was never used as a test target.
- The integration harness refuses an exact `TEST_DATABASE_URL`/`DATABASE_URL`
  match and forces all integration email through the in-memory sandbox.
- `TestAPI_EndToEndAuthAndProfileFlow`, `TestAPI_CSRFProtectionEnforcement`,
  and `TestAPI_InternalAPIKeyProtection` all passed against the isolated
  branch.
- The server route matrix and database-backed endpoint flow passed. Server
  coverage is 96.1% with the integration suite enabled; the earlier 84.2%
  figure was the no-database run.
- The initial isolated run had 2 failed integration tests and 1 panic caused
  by the email-provider test-harness defect. No production code was implicated;
  the corrected run had 0 failures and 0 panics.

## Gap-closing test added

`TestAPI_EndToEndRemainingAuthorizationMediaAndConcurrency` adds a two-user
database-backed matrix for workspace authorization and user search,
private-channel ACL enforcement, message edit/delete/reaction/read persistence,
avatar upload/finalize/attach/remove with local object storage, explicit
session-ID revocation, and concurrent session revocation. It is compiled and
executed successfully against the temporary isolated branch with no skipped
test instances.

## Coverage

Coverage figures emitted by `go test`:

| Package | Coverage |
| --- | ---: |
| auth | 43.5% |
| config | 61.7% |
| database | 46.7% |
| health | 68.2% |
| media | 53.0% |
| messaging | 18.4% |
| middleware | 56.9% |
| response | 55.9% |
| server | 96.1% |
| session | 16.2% |
| storage | 48.2% |
| users | 7.6% |
| workspaces | 13.5% |

The required cross-package command completed successfully with `coverage.out`;
the instrumented server package measured 92.1%. Dependency package percentages
remain lower because package-local store/service paths are not all exercised by
the current suite.

The requested 85% target is met for the server package (96.1%) but not for all
critical business packages. Package-level unit coverage remains low for store
implementations because server integration tests instrument the server package
and do not count dependency execution toward each dependency's own package
profile. The requested profile filename was not preserved by this Go toolchain;
the generated profile was `services/api/coverage` and was inspected with
`go tool cover`.

## Endpoint status

The route matrix verifies that each listed protected endpoint is registered and
rejects unauthenticated requests. The isolated end-to-end flow additionally
verifies the successful database-backed paths marked below. Routes not reached
by that flow remain explicitly unverified rather than being inferred from
registration alone.

| Endpoint | Registration/auth status | Database flow |
| --- | --- | --- |
| `GET /health`, `/health/live`, `/health/ready` | PASS | PASS without DB for liveness; readiness behavior covered |
| `GET /v1/health`, `/v1/health/live`, `/v1/health/ready` | PASS | PASS without DB for liveness; readiness behavior covered |
| `POST /v1/auth/otp/challenges` | PASS; malformed input covered | PASS; persisted and sandbox-delivered |
| `POST /v1/auth/otp/verifications` | PASS; malformed input covered | PASS; session persisted |
| `GET /v1/auth/google/start` | PASS; unavailable-config error covered | OAuth success blocked |
| `GET /v1/auth/google/callback` | PASS; invalid callback covered | OAuth exchange blocked |
| `GET /v1/auth/session` | PASS; protected | PASS; session flow exercised |
| `POST /v1/auth/logout` | PASS; protected | PASS; revocation verified |
| `POST /v1/media/uploads` | PASS; protected | PASS; isolated E2E |
| `POST /v1/media/{id}/finalize` | PASS; protected | PASS; isolated E2E |
| `GET /v1/media/{id}` | PASS; protected | NOT REACHED in E2E |
| `DELETE /v1/media/{id}` | PASS; protected | NOT REACHED in E2E |
| `PUT /v1/media/content/upload` | PASS; validation covered | PASS for local provider path |
| `GET /v1/media/content/*` | PASS; missing-content path covered | PASS for local provider path |
| `GET /v1/me` | PASS; protected | PASS; persisted user loaded |
| `GET /v1/me/profile` | PASS; protected | PASS; persisted profile loaded |
| `PATCH /v1/me/profile` | PASS; validation and revision checks covered | PASS; profile persistence verified |
| `POST /v1/me/onboarding/complete` | PASS; protected | PASS; onboarding persistence verified |
| `GET /v1/me/sessions` | PASS; protected | PASS; session listing exercised |
| `DELETE /v1/me/sessions/{session_id}` | PASS; protected | PASS; isolated E2E and revocation persistence |
| `GET /v1/users/search` | PASS; protected | PASS; isolated E2E and persistence |
| `POST /v1/users/me/avatar` | PASS; protected and IDOR service check | PASS; isolated E2E |
| `DELETE /v1/users/me/avatar` | PASS; protected | PASS; isolated E2E |
| `POST /v1/users/me/banner` | PASS; protected | BLOCKED |
| `DELETE /v1/users/me/banner` | PASS; protected | BLOCKED |
| `GET /v1/workspaces` | PASS; protected | PASS; persisted list exercised |
| `POST /v1/workspaces` | PASS; validation covered | PASS; persistence and cleanup verified |
| `GET /v1/workspaces/{id}` | PASS; protected | PASS; owner/member/outsider isolated E2E |
| `POST /v1/workspaces/{id}/channels` | PASS; validation covered | PASS; persistence verified |
| `GET /v1/workspaces/{id}/invites` | PASS; protected | PASS; persistence and usage verified |
| `POST /v1/workspaces/{id}/invites` | PASS; invalid duration/uses covered | PASS; creation and quota behavior verified |
| `DELETE /v1/workspaces/{id}/invites/{inviteID}` | PASS; protected | PASS; revocation verified |
| `GET /v1/invites/{code}` | PASS; invalid code covered | PASS; persisted preview and revoked-code behavior verified |
| `POST /v1/invites/{code}/accept` | PASS; unauthenticated rejection covered | PASS; membership behavior verified |
| `GET/POST /v1/channels/{id}/read` | PASS; protected | POST persistence and monotonic update PASS; GET remains unverified |
| `GET/PUT/DELETE /v1/channels/{id}/access*` | PASS; auth and unavailable-DB paths covered | PUT authorization PASS; GET/DELETE remain unverified |
| `GET/POST /v1/channels/{id}/messages` | PASS; auth, invalid ID, idempotency validation covered | PASS; persistence, ordering, and idempotent replay verified |
| `PATCH/DELETE /v1/channels/{id}/messages/{sequence}` | PASS; auth and invalid ID covered | PASS; isolated edit/delete E2E |
| `PUT/DELETE /v1/channels/{id}/messages/{sequence}/reactions/heart` | PASS; auth and invalid ID covered | PUT/idempotency PASS; DELETE remains unverified |
| `GET /v1/conversations` | PASS; protected | PASS; recipient inbox persistence verified |
| `POST /v1/conversations/direct` | PASS; protected | PASS; membership persistence verified |
| `GET/POST /v1/conversations/{id}/read` | PASS; protected | NOT REACHED in E2E |
| `GET/POST /v1/conversations/{id}/messages` | PASS; protected and idempotency routes | PASS; two-user delivery and replay verified |
| `PATCH/DELETE /v1/conversations/{id}/messages/{sequence}` | PASS; auth and invalid ID covered | NOT REACHED in E2E |
| `PUT/DELETE /v1/conversations/{id}/messages/{sequence}/reactions/heart` | PASS; auth and invalid ID covered | NOT REACHED in E2E |
| `GET /v1/internal/ping` | PASS; invalid/missing/valid key verified | PASS; valid key success verified |

## Areas tested

- **Authentication and OTP:** validation, OTP generation/verification service
  paths, rate limiting, session issuance, logout behavior, malformed input, and
  sandbox code capture.
- **Email:** Resend request contract and failure handling use a test HTTP
  transport; provider selection, sandbox normalization, and SMTP defaults are
  covered. No real email was delivered.
- **Sessions and security:** token parsing, verifier hashes, CSRF token binding,
  device hints, cookie behavior, request IDs, CORS, security headers, recovery,
  internal API key middleware, and route protection.
- **Users and media:** profile validation, optimistic revision conflicts,
  username conflicts, media ownership/IDOR protection, local object storage,
  image validation, and deletion behavior.
- **Workspaces and messaging:** invite validation and normalization, channel
  visibility validation, route authorization, malformed IDs, unavailable
  database behavior, and idempotency-related route checks.
- **Database:** pool configuration and health helpers were unit-tested. SQL
  store success/transaction behavior requires the isolated test database.

## Bugs and fixes

The isolated run confirmed and fixed a test-harness defect: integration tests
could select real `.env` email settings, return 503 for OTP, and dereference a
nil sandbox sender. The harness now clears external email settings, asserts
that the sandbox is active, and refuses an unsafe database URL match. A
regression test path passed after the fix.

The new isolated E2E run also confirmed and fixed a production query defect in
user search: the SQL referenced nonexistent `profiles.username` and
`profiles.avatar_url` columns. Search now uses the canonical user username and
a nullable avatar field compatible with the migrated schema. The E2E harness
also corrected request contracts and expectations for UUID idempotency headers,
private-channel not-found authorization, avatar-removal status, and bounded
PNG fixture dimensions.

The worktree contained pre-existing changes in backend files before this audit;
those changes were not reverted or attributed to this report.

## Remaining risks and production readiness

Production readiness is **not established**. Before release:

1. Add dedicated rollback assertions and true concurrent-operation tests for
   invitation usage and membership updates; this run directly verified
   concurrent session revocation, while message idempotency and invitation
   reuse remain covered by the existing isolated suite.
2. Run a separately approved email-sandbox test using provider test
   credentials if actual provider delivery evidence is required; no real
   recipient was contacted in this audit.
3. Raise critical business-package coverage toward the requested 85% target
   with package-local store/service tests, not line-only tests.
