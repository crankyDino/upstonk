# UPSTONK

ETF discovery with explicit eligibility rules. You describe an investor (country and account type) and the exposure you want (markets, sectors, asset classes, companies). The service then:
1. searches live public data sources
2. checks each candidate against codified regulatory rules
3. filters on your constraints
4. returns a ranked list where every result carries a justification, a confidence level and its data provenance

The rule the eligibility engine follows: **never guess**. When the data needed to make a call is missing, the result is `unknown` or `conditional`, never `eligible`.

## Quick start

```bash
# API on :8080
cd server && go run ./cmd/server

# GUI on :5173 (in another terminal)
cd gui && npm install && npm run dev
```

Open <http://localhost:5173>. Run the tests with `cd server && go test ./...`.

> **Live data is limited right now.** Yahoo only serves basic quotes to unauthenticated clients, so results have no AUM, TER or provider, and South African TFSA searches come back `unknown`. See [data sources](DOCS/data-sources/README.md).

## Documentation

All documentation lives in [DOCS/](DOCS/README.md):

- [Getting started](DOCS/getting-started/README.md): setup, configuration, smoke-test scripts
- [Architecture](DOCS/architecture/README.md): pipeline, providers, eligibility, ranking
- [API reference](DOCS/api/README.md): endpoints, fields, errors
- [Data sources](DOCS/data-sources/README.md): what each source provides and whether it works
- [Development](DOCS/development/README.md): repo layout, tests, GUI contract, extending rules and providers
- [Known gaps](DOCS/development/known-gaps.md)

## Disclaimer

Research tooling, not financial advice. Eligibility output is a best-effort reading of public data and codified rules. Always check with your platform or SARS before investing.
