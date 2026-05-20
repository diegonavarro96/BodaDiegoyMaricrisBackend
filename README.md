# Wedding RSVP Server

Tiny Go service that backs the RSVP form on the wedding site.
Standard library only, JSON file storage.

## Endpoints

| Method | Path                | Auth     | Purpose                          |
|--------|---------------------|----------|----------------------------------|
| POST   | `/api/rsvp`         | public   | Submit a new RSVP                |
| GET    | `/api/rsvp`         | admin    | List all RSVPs                   |
| DELETE | `/api/rsvp/{id}`    | admin    | Remove an RSVP                   |
| GET    | `/api/health`       | public   | Liveness check                   |

Admin auth is a static bearer token: `Authorization: Bearer $ADMIN_TOKEN`.

## RSVP shape

```json
{
  "id": "f3a1...",
  "name": "Jane Doe",
  "email": "jane@example.com",
  "attending": "yes",
  "vegetarian": false,
  "message": "Can't wait!",
  "createdAt": "2026-08-01T18:30:00Z"
}
```

## Environment variables

| Var              | Default                  | Notes                                    |
|------------------|--------------------------|------------------------------------------|
| `PORT`           | `8080`                   |                                          |
| `DATA_PATH`      | `data/rsvps.json`        | File is created on first run             |
| `ADMIN_TOKEN`    | _(unset → admin disabled)_ | Set this in prod. Long random string.  |
| `ALLOWED_ORIGIN` | `http://localhost:5173`  | Set to the wedding site origin in prod   |

## Run locally

```bash
# Terminal: in server/
ADMIN_TOKEN=dev-token go run .

# Frontend (separate terminal, in bodaDyM/)
npm run dev
```

The Vite dev server proxies `/api/*` to `http://localhost:8080`,
so the frontend can use relative URLs in both dev and prod.

## Build

```bash
go build -o wedding-rsvp .
```

Cross-compile for Linux from Windows:

```bash
GOOS=linux GOARCH=amd64 go build -o wedding-rsvp .
```

## Deploy

For Railway, see the Railway-specific guide added alongside this repo
(`RAILWAY.md`, pending).

For a self-hosted AWS/VPS deploy, the [`deploy/`](deploy/) folder has:

- `wedding-rsvp.service` — hardened systemd unit
- `wedding-rsvp.env.example` — env file template (PORT, DATA_PATH, ADMIN_TOKEN, ALLOWED_ORIGIN)
- `nginx.conf` — nginx site that serves the React build and proxies `/api/`
- `build-linux.ps1` — cross-compile a Linux/amd64 binary from Windows

## Data safety

- Writes use a mutex + atomic rename (`*.tmp` → real file), so a crash
  mid-write cannot corrupt the JSON.
- The file is plain JSON — back it up periodically (a cron `cp` is enough).
- `data/rsvps.json` is gitignored — guest data should never end up in git.
