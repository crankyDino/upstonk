# Deployment: one Render web service + Supabase

UPSTONK runs as **one container**: the Go server serves the API at `/api/v1` and the built GUI at `/`. Supabase provides the Postgres database and sign-in. Data collection (`cmd/ingest`) runs on your own machine against the same Supabase database; the Render instance only reads.

> **Sharing scope.** The data comes from sources whose terms only allow personal use. Keep the app **invite-only**, for family/friends and, ideally, a financial adviser checking the results. Opening it up further means revisiting that first (see [the plan](../plans/jse-data-store.md)).

## 1. Supabase

1. **Database:** copy the **session pooler** connection string (port **5432**, not the transaction pooler on 6543). Add `?sslmode=require`. That's `DATABASE_URL`. The schema is created automatically (tables live in the `upstonk` schema, which Supabase's REST API doesn't expose).
2. **Auth → Providers → Email:** enabled, with **Confirm email** on.
   - Keep **sign-ups enabled** in Supabase. Invited people create their own accounts, and UPSTONK itself refuses anyone who isn't invited (invite-only is on by default).
3. **Auth → URL Configuration:** set **Site URL** to your Render URL (e.g. `https://upstonk.onrender.com`) and add it to **Redirect URLs**, so confirmation emails link back to the app.
4. **Project Settings → API:** note the **project URL** (`SUPABASE_URL`) and the **publishable / anon key** (`SUPABASE_PUBLISHABLE_KEY`). The anon key is public by design. **Never** use the service-role key here.

## 2. Render

1. New → **Blueprint**, pointing at this repository. It reads `render.yaml`: one Docker web service, health check `/api/v1/health`.
2. Set the environment variables in the dashboard:

| Variable | Value |
|---|---|
| `DATABASE_URL` | Supabase session-pooler URL, port 5432, `sslmode=require` |
| `SUPABASE_URL` | `https://<project-ref>.supabase.co`; turns sign-in on |
| `SUPABASE_PUBLISHABLE_KEY` | the anon/publishable key |
| `ADMIN_EMAILS` | your email (comma-separated list); always admins |

`PORT` is set by Render, and the server listens on it. `WEB_DIR` is already set in the image.

3. Deploy. The start-up log should show `Store enabled`, `Sign-in required (Supabase Auth)` and `Serving the GUI from /app/web`.

**Free plan:** the service sleeps after a period of inactivity, so the first request after a pause is slow. That's harmless here because nothing runs on a schedule inside it.

## 3. First sign-in

1. Open the Render URL and choose **Have an invite? Create an account** with an email listed in `ADMIN_EMAILS`. That address is admitted without an invite.
2. Confirm your email, then sign in. **Admin** appears in the header.
3. Invite people with **Invite** in the header. Each invite is a one-time link for one email, valid for 14 days; send it however you like.

## 4. Keeping the data fresh (from your machine)

With `DATABASE_URL` in `server/.env` pointing at Supabase:

```bash
cd server
go run ./cmd/ingest jse-list     # after saving a new JSE ETF/ETN List into data/import/
go run ./cmd/ingest prices       # daily
go run ./cmd/ingest factsheets   # monthly (needs pdftotext: poppler-utils)
```

**Admin → Data status** shows each source's last run, coverage and failures.

## Local check of the production image

```bash
docker build -t upstonk .
docker run --rm -p 8080:8080 --env-file server/.env upstonk
```
