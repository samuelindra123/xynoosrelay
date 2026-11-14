# XYNOSS Relay API

Go HTTP API for authentication, profiles, sessions, media, direct messages, workspaces, channels, invitations, and access control.

## Runtime

- Go version declared by `go.mod`: 1.27.3 or newer.
- Chi router, pgx PostgreSQL pool, Neon private object storage, and Goose SQL migrations.
- Local server entry point: `cmd/server`.
- Vercel Go function entry point: `api/index.go`, configured by `vercel.json`.

## Local setup

```powershell
Copy-Item .env.example .env
# Set DATABASE_URL for persistent API features.
go run ./cmd/server
```

The local template uses filesystem media storage and local HTTP origins. To use account, messaging, workspace, or media persistence, configure a development PostgreSQL/Neon database and run:

```powershell
go run ./cmd/migrate
```

The migration runner uses the direct/unpooled database URL when configured. Keep the pooled `DATABASE_URL` for the runtime server.

## Environment

| Variable group | Purpose |
| --- | --- |
| `APP_ENV`, `PORT`, `API_BASE_URL` | Runtime mode, listener, and public API origin |
| `FRONTEND_URL`, `WEB_ORIGINS` | OAuth redirects and explicit credentialed CORS origins |
| `DATABASE_URL`, `DATABASE_DIRECT_URL` | Runtime pool and migration connection |
| `SESSION_HASH_KEY`, `OTP_HMAC_KEYS` | Session verification and versioned OTP HMAC secrets |
| `GOOGLE_CLIENT_*`, `GOOGLE_REDIRECT_URI` | Optional Google OAuth |
| `RESEND_*` or `SMTP_*` | Transactional email for email-code sign-in |
| `STORAGE_*`, `AWS_*` | Local or remote object storage. Neon production storage requires endpoint, region, bucket, and credentials. |
| `INTERNAL_API_KEYS` | Optional server-to-server credentials; never expose to the browser |

Copy `services/api/.env.production.example` as a checklist only. Replace every example hostname and configure actual credentials in the deployment provider's secret manager. The repository's local `.env` is ignored and must remain separate from production secrets. Production/staging startup requires HTTPS API/frontend origins, matching explicit CORS origins, PostgreSQL, Neon storage, session and OTP keys, and an email provider.

## Browser media delivery

Remote object storage is private. Profile media URLs use `GET /v1/media/profile/{media_id}`; the endpoint returns an image only when its finalized media ID is currently attached as an avatar or banner. It never returns storage credentials or makes pending/unattached uploads public. Other media management routes remain authenticated.

## Health checks

- `GET /health/live` — process liveness.
- `GET /health/ready` — database readiness.
- `GET /v1/health` — API status.

Errors use a consistent JSON envelope with code, message, details, and request ID. Production logs include request metadata (method, path, status, duration, remote address, and user agent); request bodies and credentials are not logged by the HTTP request logger.

## Checks

Run from `services/api`:

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test ./...
go build ./...
```

## Vercel

Deploy this directory as the API project's root. Set the environment variables above in Vercel for Production, Preview, and Development as appropriate. Do not rely on the Vercel function's local filesystem for persistent uploads. A configuration error returns `503`; the function does not silently fall back to an in-memory or local-storage production server.
