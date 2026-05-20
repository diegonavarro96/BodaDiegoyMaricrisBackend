# Deploying the RSVP backend to Railway

This is the Go service that stores RSVPs as JSON. It's deployed as a
separate Railway service from the React frontend
([`BodaDiegoyMaricris`](https://github.com/diegonavarro96/BodaDiegoyMaricris)).

## One-time setup

1. **New service from GitHub** → choose `BodaDiegoyMaricrisBackend` →
   branch `main`. Railway detects Go from `go.mod` and uses
   [`railway.json`](railway.json) for healthcheck and restart policy.

2. **Add a volume** so the JSON store survives redeploys.
   Service → Settings → Volumes → New Volume:
   - Mount path: `/data`
   - Size: 1 GB is plenty (RSVPs are tiny)

3. **Set env vars** (Service → Variables):

   | Var              | Value                                                                 |
   |------------------|-----------------------------------------------------------------------|
   | `ADMIN_TOKEN`    | long random string (`openssl rand -hex 32`)                           |
   | `ALLOWED_ORIGIN` | the frontend's Railway URL, no trailing slash, e.g. `https://boda.up.railway.app` |
   | `DATA_PATH`      | `/data/rsvps.json`  *(must match the volume mount path)*              |
   | `PORT`           | leave **unset** — Railway injects it automatically                    |

4. **Generate a public domain** (Service → Networking → Generate Domain).
   Copy the URL — paste it into the frontend's `VITE_API_URL` env var and
   redeploy the frontend.

5. **Smoke test** once it's up:
   ```bash
   curl https://your-backend.up.railway.app/api/health
   # → {"status":"ok"}
   ```

## Why the volume matters

The service stores RSVPs in a JSON file (`$DATA_PATH`). Without a mounted
volume, that file lives on the ephemeral container filesystem and **every
redeploy wipes guest responses**. The volume persists it across deploys
and restarts.

## CORS

`ALLOWED_ORIGIN` must exactly match the frontend's scheme + host (no path,
no trailing slash). If you later attach a custom domain to the frontend,
update `ALLOWED_ORIGIN` to match and restart this service.

## Admin

Visit `https://your-frontend-domain/#admin` and paste `ADMIN_TOKEN`. The
dashboard lists/deletes RSVPs and exports CSV.

## Local development

```bash
ADMIN_TOKEN=dev-token go run .
# in another terminal, in bodaDyM/: npm run dev
```

## Updates

Pushing to `main` triggers a Railway redeploy. Watch the deploy logs in the
dashboard; the healthcheck on `/api/health` must pass for traffic to cut
over.
