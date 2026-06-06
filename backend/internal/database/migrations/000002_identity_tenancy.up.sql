-- Shop (tenant unit)
CREATE TABLE shop (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL,
    phone      TEXT,
    address    TEXT,
    city       TEXT,
    state      CHAR(2),
    timezone   TEXT        NOT NULL DEFAULT 'America/Sao_Paulo',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT shop_slug_unique UNIQUE (slug)
);

-- Role enum
CREATE TYPE user_role AS ENUM ('owner', 'barber', 'customer');

-- User (global; one account can belong to many shops with different roles)
-- Note: "user" is a reserved keyword in PostgreSQL and must always be quoted.
CREATE TABLE "user" (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    phone         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_email_unique UNIQUE (email)
);

-- Membership: links a user to a shop with a role (one role per user per shop)
CREATE TABLE membership (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id    UUID        NOT NULL REFERENCES shop   (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    role       user_role   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT membership_shop_user_unique UNIQUE (shop_id, user_id)
);

CREATE INDEX membership_shop_id_idx ON membership (shop_id);
CREATE INDEX membership_user_id_idx ON membership (user_id);
