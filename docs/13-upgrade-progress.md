# Discord-style messenger upgrade

## Implemented in this change

- DM and public-channel message history accept `before_sequence` (exclusive positive int64) and `limit` (1–100, default 100). Existing array responses remain compatible. Query authorization still runs before reading any messages.
- UI loads older history, deduplicates overlapping pages, retains older pages during polling, and preserves the scroll position when prepending messages.
- `GET` and `POST /v1/channels/{id}/read` and `/v1/conversations/{id}/read` persist read cursors. POST accepts `{ "last_read_sequence": 25 }`. Updates cannot move backwards or exceed the conversation head. Access rows are locked until commit.
- UI restores server read state when opening a conversation and updates its cursor only after server acknowledgement. Old browser-only read cursors no longer drive unread state.
- Remote storage configuration requires bucket and credentials; construction failure never silently falls back to local disk.
- Public/private channel visibility is stored and shown in the UI. Workspace listings filter private channels by owner or explicit `channel.read` user ACL; channel reads, sends, and read receipts use the same access rule. New private channels grant access to their creator and workspace owner.
- Workspace owners can manage private-channel grants from the messenger UI. Granting access requires active workspace membership; leaving and rejoining clears old private-channel grants so they must be explicitly restored.
- DM and channel messages support server-side edit with revision conflicts, author-only soft deletion, and idempotent heart reactions. Controls in the chat call these APIs.
- App-level error, global error, and custom 404 screens use friendly Indonesian copy and recovery links without rendering exception text or backend responses.
- API failures are normalized into safe messages. Production rejects missing, malformed, and loopback API URLs instead of silently requesting `localhost`; development retains its local default.
- Invitation preview and acceptance errors use safe copy; temporary connection/server failures offer a retry action. Invite metadata falls back safely when the API URL is unavailable.
- Google sign-in checks backend/provider availability before navigating away; offline or unconfigured states stay on the login page with a friendly message.

## Validation and limits

Frontend unit tests exercise history merging, retry identity, read visibility, logout and safe redirects. Backend unit tests cover cursor parsing, read payload validation and remote storage configuration. These unit tests do not prove PostgreSQL concurrency or browser integration. Database integration tests require an explicitly configured isolated `TEST_DATABASE_URL`. Production readiness requires those integration checks and staging evidence.

No development servers are needed for these checks. Headless browser verification requires an available Chrome/Chromium installation and a served production build or a dedicated browser fixture.

Validation performed: all 18 frontend unit tests passed; TypeScript, ESLint, and the optimized Next.js production build passed; `go test ./...` passed across the API module. Database integration tests were skipped because `TEST_DATABASE_URL` was not configured. No browser pass or full application E2E pass is claimed; this pass respects the request to avoid headless browser use.

## Remaining implementation order

1. Generalize per-user private-channel grants to role-based ACLs, then add PostgreSQL authorization race/concurrency tests.
2. Durable replies, pins, reaction summaries, permission-filtered search and attachments.
3. Outbox publication, authorized realtime subscriptions, reconnect reconciliation, presence and typing.
4. Notifications, blocking/reporting/moderation and account privacy controls.
5. Split the messenger UI into feature components; improve composer, mobile navigation, keyboard access, themes and empty/error/offline states.
6. CI security checks, PostgreSQL concurrency tests, production build browser checks, load tests, monitoring, restore drills and release runbooks.

Voice/video, bots, group DMs and threads require separate product and infrastructure work. This change completes an initial messaging slice and does not imply Discord feature parity or production readiness.
