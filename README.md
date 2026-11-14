# XYNOSS Relay

**Messaging without the weight. Less app. More conversation.**

XYNOSS Relay is a conversation product with a Next.js client and a Go API. This repository contains the product website and an app interface for direct messages, workspaces, channels, profiles, sessions, invitations, and media.

## Project map

| Path | Responsibility |
| --- | --- |
| `apps/client` | Next.js 16 App Router, React 19, TypeScript, responsive interface, and API client |
| `services/api` | Go 1.27 HTTP API, PostgreSQL persistence, authentication, media, workspaces, and messaging |
| `services/api/migrations` | PostgreSQL schema migrations |
| `neon.ts` | Neon branch, database, auth, realtime, and private upload bucket configuration |

## Product capabilities

- Google OAuth and email one-time-code authentication with session cookies, CSRF protection, and Bearer support.
- Direct conversations and workspace channels with history, edits, deletion, reactions, read markers, and retry-safe sends.
- User profiles with bio, availability, custom status, avatar and banner crop, upload progress, and removal.
- Workspace membership, private channel access, invitations, and active session management.
- Private object storage with a profile-media endpoint that serves only finalized images attached to a public profile.
- Indonesian interface copy, light and dark themes, responsive layouts, keyboard focus states, and accessible dialog flows.

Feature availability depends on the backend and production environment being configured. See the service documentation and environment templates before deployment.

## Local development

Requirements: Node.js supported by Next.js 16, pnpm 12.18 or compatible, and Go 1.27.3 or newer.

### Web client

```powershell
cd apps/client
pnpm install
Copy-Item .env.example .env.local
pnpm dev
```

The example points to `http://localhost:8080/v1`. Start the API before using sign-in or server-backed app features.

### Go API

```powershell
cd services/api
Copy-Item .env.example .env
# Set DATABASE_URL and any required local provider credentials in .env.
go run ./cmd/server
```

The API listens on `http://localhost:8080` by default. For a connected database, apply migrations with `go run ./cmd/migrate` before using persistent features. Do not copy production credentials into a local file.

## Quality checks

Run from `apps/client`:

```powershell
pnpm test
pnpm lint
pnpm build
```

Run from `services/api`:

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test ./...
go build ./...
```

## Deployment configuration

- Deploy the Next.js client from `apps/client` and set `NEXT_PUBLIC_API_URL` to the public API URL ending in `/v1`.
- Deploy the Go API from `services/api` with the included Vercel entry point or run `cmd/server` in a Go container.
- Configure matching HTTPS values for `API_BASE_URL`, `FRONTEND_URL`, `WEB_ORIGINS`, and the Google OAuth callback.
- Use a Neon production branch, private object-storage bucket, unique session and OTP secrets, and a verified transactional email sender.
- `services/api/.env.production.example` is a template, not a deployable secret file. Put real values in the hosting provider's environment/secret manager. The local `services/api/.env` is ignored and intentionally remains a development configuration.
- Never put `INTERNAL_API_KEYS`, database credentials, OAuth secrets, storage credentials, or other backend secrets in client code or any `NEXT_PUBLIC_*` variable.

Production and staging configuration fails closed unless required storage, database, email, and HTTPS settings are present. Review deployment logs and the readiness endpoint after configuration.

## Privacy and legal pages

The client includes privacy and terms pages that describe implemented data flows and user controls. The operator's legal identity, official privacy contact, and infrastructure retention periods still need confirmation before the pages can be treated as final legal documents.

## License

No open-source license is currently declared. All rights remain with the project owner unless a license is added.
