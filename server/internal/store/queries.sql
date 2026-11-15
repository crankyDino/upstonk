-- Writes (used by the ingest jobs and by tests to set up data)

-- name: UpsertETF :exec
INSERT INTO upstonk.etfs (code, name, isin, issuer, exchange, currency,
                          listing_source, listing_url, listed_as_of, active, updated_at)
VALUES (@code, @name, @isin, @issuer, @exchange, @currency,
        @listing_source, @listing_url, @listed_as_of, true, now())
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name, isin = EXCLUDED.isin, issuer = EXCLUDED.issuer,
    exchange = EXCLUDED.exchange, currency = EXCLUDED.currency,
    listing_source = EXCLUDED.listing_source, listing_url = EXCLUDED.listing_url,
    listed_as_of = EXCLUDED.listed_as_of, active = true, updated_at = now();

-- name: InsertFactsheet :one
-- Re-reading the same document (same ETF, date and URL) replaces its values.
INSERT INTO upstonk.factsheets (etf_code, as_of, source_name, source_url,
    ter_percent, aum, aum_currency, tracked_index, asset_class, domicile,
    legal_structure, replication, is_leveraged, is_inverse,
    distribution_policy, inception_date, tfsa_eligible)
VALUES (@etf_code, @as_of, @source_name, @source_url,
    @ter_percent, @aum, @aum_currency, @tracked_index, @asset_class, @domicile,
    @legal_structure, @replication, @is_leveraged, @is_inverse,
    @distribution_policy, @inception_date, @tfsa_eligible)
ON CONFLICT (etf_code, as_of, source_url) DO UPDATE SET
    source_name = EXCLUDED.source_name, fetched_at = now(),
    ter_percent = EXCLUDED.ter_percent, aum = EXCLUDED.aum, aum_currency = EXCLUDED.aum_currency,
    tracked_index = EXCLUDED.tracked_index, asset_class = EXCLUDED.asset_class,
    domicile = EXCLUDED.domicile, legal_structure = EXCLUDED.legal_structure,
    replication = EXCLUDED.replication, is_leveraged = EXCLUDED.is_leveraged,
    is_inverse = EXCLUDED.is_inverse, distribution_policy = EXCLUDED.distribution_policy,
    inception_date = EXCLUDED.inception_date, tfsa_eligible = EXCLUDED.tfsa_eligible
RETURNING id;

-- Reads

-- name: ListActiveETFs :many
-- Active ETFs only (ETNs are stored but not yet searchable).
SELECT code, name, isin, issuer, exchange, currency,
       listing_source, listing_url, listed_as_of,
       tfsa_friendly, underlying_index, category
FROM upstonk.etfs
WHERE active AND instrument_type = 'ETF'
ORDER BY code;

-- name: ListLatestFactsheets :many
-- The most recent factsheet per ETF (by stated date, then fetch time).
SELECT DISTINCT ON (etf_code)
       id, etf_code, as_of, source_name, source_url,
       ter_percent, aum, aum_currency, tracked_index, asset_class, domicile,
       legal_structure, replication, is_leveraged, is_inverse,
       distribution_policy, inception_date, tfsa_eligible
FROM upstonk.factsheets
ORDER BY etf_code, as_of DESC, fetched_at DESC;

-- name: UpsertDailyPrice :exec
-- One row per ETF per trading day; re-fetching a day replaces it.
INSERT INTO upstonk.daily_prices (etf_code, trade_date, close, currency, volume, source_name, source_url)
VALUES (@etf_code, @trade_date, @close, @currency, @volume, @source_name, @source_url)
ON CONFLICT (etf_code, trade_date) DO UPDATE SET
    close = EXCLUDED.close, currency = EXCLUDED.currency, volume = EXCLUDED.volume,
    source_name = EXCLUDED.source_name, source_url = EXCLUDED.source_url, fetched_at = now();

-- name: ListVolumeStats :many
-- Average daily volume over the 3 months up to each ETF's latest stored day,
-- with the source and date range it came from. Days without volume don't count.
SELECT p.etf_code,
       avg(p.volume)::float8      AS avg_volume,
       count(p.volume)::int       AS days,
       min(p.trade_date)::date    AS from_date,
       max(p.trade_date)::date    AS to_date,
       max(p.source_name)::text   AS source_name,
       max(p.source_url)::text    AS source_url
FROM upstonk.daily_prices p
JOIN (SELECT etf_code, max(trade_date) AS last_day FROM upstonk.daily_prices GROUP BY etf_code) l
  ON l.etf_code = p.etf_code
WHERE p.trade_date > l.last_day - interval '3 months' AND p.volume IS NOT NULL
GROUP BY p.etf_code;

-- name: ListDailyPrices :many
SELECT etf_code, trade_date, close, currency, volume, source_name, source_url, fetched_at
FROM upstonk.daily_prices
WHERE etf_code = @etf_code
ORDER BY trade_date;

-- name: DeleteFactsheetDetails :exec
-- Clears a factsheet's holdings and exposures before re-writing them.
DELETE FROM upstonk.factsheet_holdings WHERE factsheet_id = @factsheet_id;

-- name: DeleteFactsheetExposures :exec
DELETE FROM upstonk.factsheet_exposures WHERE factsheet_id = @factsheet_id;

-- name: InsertHolding :exec
INSERT INTO upstonk.factsheet_holdings (factsheet_id, position, name, ticker, weight_percent)
VALUES (@factsheet_id, @position, @name, @ticker, @weight_percent);

-- name: InsertExposure :exec
INSERT INTO upstonk.factsheet_exposures (factsheet_id, kind, name, weight_percent)
VALUES (@factsheet_id, @kind, @name, @weight_percent);

-- name: ListHoldings :many
SELECT position, name, ticker, weight_percent
FROM upstonk.factsheet_holdings WHERE factsheet_id = @factsheet_id ORDER BY position;

-- name: ListExposures :many
SELECT kind, name, weight_percent
FROM upstonk.factsheet_exposures WHERE factsheet_id = @factsheet_id ORDER BY kind, weight_percent DESC;

-- name: ListHoldingsFor :many
SELECT factsheet_id, position, name, ticker, weight_percent
FROM upstonk.factsheet_holdings
WHERE factsheet_id = ANY(@factsheet_ids::bigint[])
ORDER BY factsheet_id, position;

-- name: ListExposuresFor :many
SELECT factsheet_id, kind, name, weight_percent
FROM upstonk.factsheet_exposures
WHERE factsheet_id = ANY(@factsheet_ids::bigint[])
ORDER BY factsheet_id, kind, weight_percent DESC;

-- name: UpsertListing :exec
-- One row of a JSE ETF/ETN List import. Re-imports update in place and
-- re-activate; the factsheet-only columns (isin, exchange) are left alone.
INSERT INTO upstonk.etfs (code, name, issuer, issuer_url, exchange, currency, instrument_type,
    category, underlying_index, tfsa_friendly, foreign_status, sarb_classification,
    disclosure_frequency, listing_source, listing_url, listed_as_of, active, updated_at)
VALUES (@code, @name, @issuer, @issuer_url, 'JSE', @currency, @instrument_type,
    @category, @underlying_index, @tfsa_friendly, @foreign_status, @sarb_classification,
    @disclosure_frequency, @listing_source, @listing_url, @listed_as_of, true, now())
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name, issuer = EXCLUDED.issuer, issuer_url = EXCLUDED.issuer_url,
    currency = EXCLUDED.currency, instrument_type = EXCLUDED.instrument_type,
    category = EXCLUDED.category, underlying_index = EXCLUDED.underlying_index,
    tfsa_friendly = EXCLUDED.tfsa_friendly, foreign_status = EXCLUDED.foreign_status,
    sarb_classification = EXCLUDED.sarb_classification,
    disclosure_frequency = EXCLUDED.disclosure_frequency,
    listing_source = EXCLUDED.listing_source, listing_url = EXCLUDED.listing_url,
    listed_as_of = EXCLUDED.listed_as_of, active = true, updated_at = now();

-- name: DeactivateMissing :execrows
-- After a full list import: instruments of that type not in the new list
-- are no longer listed.
UPDATE upstonk.etfs SET active = false, updated_at = now()
WHERE instrument_type = @instrument_type AND active AND NOT (code = ANY(@codes::text[]));

-- name: ListListings :many
SELECT code, name, issuer, issuer_url, currency, instrument_type, category, underlying_index,
       tfsa_friendly, foreign_status, sarb_classification, disclosure_frequency,
       listing_source, listing_url, listed_as_of, active
FROM upstonk.etfs
ORDER BY code;

-- Auth and membership

-- name: GetAppUser :one
SELECT id, email, role, status, invited_by, created_at, last_seen_at
FROM upstonk.app_users WHERE id = @id;

-- name: InsertAppUser :exec
-- First admission; a concurrent first request may have inserted already.
INSERT INTO upstonk.app_users (id, email, role, invited_by)
VALUES (@id, @email, @role, @invited_by)
ON CONFLICT (id) DO NOTHING;

-- name: TouchAppUser :exec
UPDATE upstonk.app_users SET email = @email, last_seen_at = now() WHERE id = @id;

-- name: SetAppUserRole :exec
UPDATE upstonk.app_users SET role = @role WHERE id = @id;

-- name: SetAppUserStatus :exec
UPDATE upstonk.app_users SET status = @status WHERE id = @id;

-- name: ListAppUsers :many
SELECT u.id, u.email, u.role, u.status, u.invited_by, i.email AS invited_by_email,
       u.created_at, u.last_seen_at
FROM upstonk.app_users u
LEFT JOIN upstonk.app_users i ON i.id = u.invited_by
ORDER BY u.created_at;

-- name: CountActiveAdmins :one
SELECT count(*)::int FROM upstonk.app_users WHERE role = 'admin' AND status = 'active';

-- name: GetSettings :one
SELECT invite_only, notice, updated_at, updated_by FROM upstonk.app_settings WHERE id = 1;

-- name: UpdateSettings :exec
UPDATE upstonk.app_settings
SET invite_only = @invite_only, notice = @notice, updated_at = now(), updated_by = @updated_by
WHERE id = 1;

-- name: CreateInvitation :one
INSERT INTO upstonk.invitations (email, token, invited_by, expires_at)
VALUES (lower(@email), @token, @invited_by, @expires_at)
RETURNING id, email, token, invited_by, created_at, expires_at;

-- name: FindOpenInvitation :one
-- An unused, unrevoked, unexpired invitation matching both link token and email.
SELECT id, email, invited_by FROM upstonk.invitations
WHERE token = @token AND email = lower(@email)
  AND used_at IS NULL AND revoked_at IS NULL AND expires_at > now();

-- name: UseInvitation :exec
UPDATE upstonk.invitations SET used_at = now(), used_by = @used_by WHERE id = @id AND used_at IS NULL;

-- name: RevokeInvitation :execrows
UPDATE upstonk.invitations SET revoked_at = now()
WHERE id = @id AND used_at IS NULL AND revoked_at IS NULL;

-- name: ListInvitations :many
SELECT i.id, i.email, i.invited_by, u.email AS invited_by_email, i.created_at, i.expires_at,
       i.used_at, i.used_by, i.revoked_at
FROM upstonk.invitations i
JOIN upstonk.app_users u ON u.id = i.invited_by
ORDER BY i.created_at DESC;
