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

## Deploy notes (AWS)

1. `scp` the binary and create `/var/www/wedding-rsvp/`.
2. Put `data/` next to the binary; the directory must be writable by the service user.
3. Run under systemd. Example unit:

   ```ini
   [Service]
   Environment=ADMIN_TOKEN=<long-random-string>
   Environment=ALLOWED_ORIGIN=https://your-wedding-domain
   WorkingDirectory=/var/www/wedding-rsvp
   ExecStart=/var/www/wedding-rsvp/wedding-rsvp
   Restart=always
   ```

4. Reverse-proxy `/api/` to `127.0.0.1:8080` from nginx so the API and
   static site share the same origin and HTTPS cert.

## Data safety

- Writes use a mutex + atomic rename (`*.tmp` → real file), so a crash
  mid-write cannot corrupt the JSON.
- The file is plain JSON — back it up periodically (a cron `cp` is enough).
- `data/rsvps.json` is gitignored — guest data should never end up in git.
