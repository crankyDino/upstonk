# Getting started

## Prerequisites

- **Go 1.25+** (the module declares `go 1.25.5`)
- **Node 18+** with npm. `npm run contract:fixture` runs TypeScript directly, which needs Node 22.18+ or 23.6+ (checked with Node 24).
- Optional: [`air`](https://github.com/air-verse/air) for Go live reload, and `jq` for the smoke-test scripts

## 1. Run the API

```bash
cd server
go mod download
go run ./cmd/server        # listens on :8080
# or, with live reload (config in server/.air.toml):
air
```

Check that it's up:

```bash
curl http://localhost:8080/api/v1/health
# {"service":"upstonk-api","status":"healthy","timestamp":"…","version":"1.0.0"}
```

`GET /` serves a small HTML page describing the API.

## 2. Run the GUI

```bash
cd gui
npm install
npm run dev     # http://localhost:5173
```

The GUI calls `http://localhost:8080/api/v1` by default. If the API runs somewhere else, set `VITE_UPSTONK_API_URL` in `gui/.env`.

Other scripts: `npm run build`, `npm run build:dev`, `npm run preview`, `npm run lint`, `npm run contract:fixture` (see [development](../development/README.md#gui--server-contract)).

## Configuration

### Server

Read from environment variables. A `server/.env` file is optional and is loaded if present.

| Variable | Description | Default |
|---|---|---|
| `SERVER_ADDRESS` | Bind address | `:8080` |
| `ALPHA_VANTAGE_API_KEY` | Enables the Alpha Vantage provider. The literal `demo` leaves it disabled. | `demo` |
| `DATABASE_URL` | Postgres store (Supabase **session** pooler, port 5432, `sslmode=require`). When set, the server applies migrations at start-up and answers ZA searches from stored data. If it's set but unreachable, the server won't start. | empty (no store) |

| `SUPABASE_URL` | Turns on sign-in (Supabase Auth): every API route except `/api/v1/health` needs a signed-in, admitted user. Requires `DATABASE_URL`. | empty (API open) |
| `SUPABASE_PUBLISHABLE_KEY` | Public anon key, handed to the browser via `/config.js` | empty |
| `ADMIN_EMAILS` | Comma-separated emails that are always admins | empty |
| `WEB_DIR` | Built GUI (`gui/dist`) to serve at `/` (single instance) | empty |
| `PORT` | Used when `SERVER_ADDRESS` isn't set (Render sets it) | — |

`server/.env.example` lists them. For hosting, see [deployment](../deployment/README.md). The search cache is fixed at one hour in memory, and logging uses Go's standard `log` package.

### GUI (`gui/.env`)

| Variable | Description | Default |
|---|---|---|
| `VITE_UPSTONK_API_URL` | Full API base URL, including `/api/v1` | `http://localhost:8080/api/v1` |

## Filling the store (JSE data)

Personal use only: requests go one at a time, seconds apart (see [the plan](../plans/jse-data-store.md)).

```bash
cd server
go run ./cmd/ingest prices       # daily: ~3 months of Yahoo closes and volumes for every stored ETF
go run ./cmd/ingest factsheets   # monthly: issuer factsheets (Satrix so far)
```

The list of ETFs comes from the JSE's "ETF List" download, which you save yourself because the JSE site shows a bot challenge. The importer for it is still to come.

## Smoke-test scripts

Both scripts need a running API and default to `http://localhost:8080`. Override with `API_URL`.

```bash
cd server
./scripts/test_live_data.sh       # health, US tech, emerging markets, cache timing (needs jq)
./examples/test_payloads.sh       # four discovery payloads, raw JSON output
API_URL=http://localhost:9000 ./scripts/test_live_data.sh
```

## What to expect from live data today

Yahoo now only serves basic quotes to unauthenticated clients. Results therefore have no AUM, TER, provider or holdings. As a consequence:

- **South African TFSA searches return no `results`.** Eligibility can't be confirmed without the provider, so every fund comes back `unknown`, and `tfsaEligibleOnly: true` (the GUI's default) drops unknowns. Set `tfsaEligibleOnly: false` with `includeAlternatives: true` to see the candidates.
- Standard-account searches work. Funds with unknown AUM or TER are kept under `minAUM`/`maxTER` and flagged with `UNVERIFIED_AUM` / `UNVERIFIED_TER` warnings.

Details are in [data-sources](../data-sources/README.md) and [known gaps](../development/known-gaps.md).
