-- +goose Up
-- All UPSTONK data lives in its own schema so Supabase's REST API (which
-- serves "public") doesn't expose it. Every collected value records its
-- source; a value no source reported stays NULL, never a default.

CREATE SCHEMA IF NOT EXISTS upstonk;

CREATE TABLE IF NOT EXISTS upstonk.etfs (
    code            text PRIMARY KEY,          -- JSE code, e.g. STX40
    name            text NOT NULL,
    isin            text,
    issuer          text,
    exchange        text NOT NULL DEFAULT 'JSE',
    currency        text,
    active          boolean NOT NULL DEFAULT true,
    listing_source  text NOT NULL,             -- where the listing came from
    listing_url     text,
    listed_as_of    date NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- One row per factsheet (or MDD) read; history is kept.
CREATE TABLE IF NOT EXISTS upstonk.factsheets (
    id                  bigserial PRIMARY KEY,
    etf_code            text NOT NULL REFERENCES upstonk.etfs(code) ON DELETE CASCADE,
    as_of               date NOT NULL,         -- date the document states
    source_name         text NOT NULL,
    source_url          text NOT NULL,
    fetched_at          timestamptz NOT NULL DEFAULT now(),
    ter_percent         numeric,               -- e.g. 0.10 = 0.10%
    aum                 numeric,
    aum_currency        text,
    tracked_index       text,
    asset_class         text,
    domicile            text,
    legal_structure     text,
    replication         text CHECK (replication IN ('physical', 'synthetic')),
    is_leveraged        boolean,
    is_inverse          boolean,
    distribution_policy text,
    inception_date      date,
    tfsa_eligible       boolean,
    UNIQUE (etf_code, as_of, source_url)
);

CREATE TABLE IF NOT EXISTS upstonk.factsheet_holdings (
    factsheet_id   bigint NOT NULL REFERENCES upstonk.factsheets(id) ON DELETE CASCADE,
    position       int NOT NULL,
    name           text NOT NULL,
    ticker         text,
    weight_percent numeric,
    PRIMARY KEY (factsheet_id, position)
);

CREATE TABLE IF NOT EXISTS upstonk.factsheet_exposures (
    factsheet_id   bigint NOT NULL REFERENCES upstonk.factsheets(id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('sector', 'region', 'country')),
    name           text NOT NULL,
    weight_percent numeric NOT NULL,
    PRIMARY KEY (factsheet_id, kind, name)
);

CREATE TABLE IF NOT EXISTS upstonk.daily_prices (
    etf_code     text NOT NULL REFERENCES upstonk.etfs(code) ON DELETE CASCADE,
    trade_date   date NOT NULL,
    close        numeric,                      -- in currency below (major units)
    currency     text,
    volume       bigint,
    source_name  text NOT NULL,
    source_url   text NOT NULL,
    fetched_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (etf_code, trade_date)
);

CREATE INDEX IF NOT EXISTS factsheets_latest ON upstonk.factsheets (etf_code, as_of DESC);

-- +goose Down
DROP TABLE IF EXISTS upstonk.daily_prices;
DROP TABLE IF EXISTS upstonk.factsheet_exposures;
DROP TABLE IF EXISTS upstonk.factsheet_holdings;
DROP TABLE IF EXISTS upstonk.factsheets;
DROP TABLE IF EXISTS upstonk.etfs;
