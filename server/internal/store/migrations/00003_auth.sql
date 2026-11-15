-- +goose Up
-- App membership on top of Supabase Auth. Supabase owns identities
-- (auth.users); these tables decide who may use UPSTONK and how.

CREATE TABLE IF NOT EXISTS upstonk.app_users (
    id          uuid PRIMARY KEY,                 -- Supabase auth user id (JWT "sub")
    email       text NOT NULL,
    role        text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    invited_by  uuid REFERENCES upstonk.app_users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz
);

CREATE TABLE IF NOT EXISTS upstonk.invitations (
    id          bigserial PRIMARY KEY,
    email       text NOT NULL,                    -- stored lower-case
    token       text NOT NULL UNIQUE,             -- secret in the invite link
    invited_by  uuid NOT NULL REFERENCES upstonk.app_users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    used_by     uuid REFERENCES upstonk.app_users(id) ON DELETE SET NULL,
    revoked_at  timestamptz
);

-- One row of app-wide settings.
CREATE TABLE IF NOT EXISTS upstonk.app_settings (
    id          int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    invite_only boolean NOT NULL DEFAULT true,
    notice      text,                             -- banner shown to all users
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  uuid REFERENCES upstonk.app_users(id) ON DELETE SET NULL
);
INSERT INTO upstonk.app_settings (id) VALUES (1) ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS upstonk.app_settings;
DROP TABLE IF EXISTS upstonk.invitations;
DROP TABLE IF EXISTS upstonk.app_users;
