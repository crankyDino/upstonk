-- +goose Up
-- Facts the JSE publishes in its ETF/ETN List downloads. TFSA, foreign and
-- SARB fields are the JSE's (from FSCA/issuer information, per the list's
-- disclaimer); NULL when the list doesn't state them.
ALTER TABLE upstonk.etfs
    ADD COLUMN IF NOT EXISTS instrument_type      text NOT NULL DEFAULT 'ETF'
        CHECK (instrument_type IN ('ETF', 'ETN')),
    ADD COLUMN IF NOT EXISTS category             text,
    ADD COLUMN IF NOT EXISTS underlying_index     text,
    ADD COLUMN IF NOT EXISTS tfsa_friendly        boolean,
    ADD COLUMN IF NOT EXISTS foreign_status       boolean,
    ADD COLUMN IF NOT EXISTS sarb_classification  text,
    ADD COLUMN IF NOT EXISTS disclosure_frequency text,
    ADD COLUMN IF NOT EXISTS issuer_url           text;

-- +goose Down
ALTER TABLE upstonk.etfs
    DROP COLUMN IF EXISTS issuer_url,
    DROP COLUMN IF EXISTS disclosure_frequency,
    DROP COLUMN IF EXISTS sarb_classification,
    DROP COLUMN IF EXISTS foreign_status,
    DROP COLUMN IF EXISTS tfsa_friendly,
    DROP COLUMN IF EXISTS underlying_index,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS instrument_type;
