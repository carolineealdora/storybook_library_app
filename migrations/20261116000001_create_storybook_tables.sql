-- Schema for the Digital Storybook Library + Book Management apps (PRD §12.2).
-- Money is INTEGER IDR. Enums are VARCHAR + CHECK. Files (EPUB, covers) live in the DB.

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL,
    pin_hash            TEXT,
    pin_failed_attempts INT NOT NULL DEFAULT 0,
    pin_locked_until    TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE admins (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE credentials (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    account_type  VARCHAR(10) NOT NULL CHECK (account_type IN ('parent', 'admin')),
    user_id       UUID REFERENCES users (id),
    admin_id      UUID REFERENCES admins (id),
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((account_type = 'parent' AND user_id IS NOT NULL AND admin_id IS NULL)
        OR (account_type = 'admin' AND admin_id IS NOT NULL AND user_id IS NULL))
);
CREATE UNIQUE INDEX credentials_email_key ON credentials (lower(email));
CREATE UNIQUE INDEX credentials_user_id_key ON credentials (user_id);
CREATE UNIQUE INDEX credentials_admin_id_key ON credentials (admin_id);

CREATE TABLE refresh_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES credentials (id),
    token_hash    CHAR(64) NOT NULL UNIQUE,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_credential_id_idx ON refresh_tokens (credential_id);

CREATE TABLE books (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR(255) NOT NULL,
    author          VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL,
    age_type        VARCHAR(10) NOT NULL CHECK (age_type IN ('range', 'plus')),
    age_min         INT NOT NULL CHECK (age_min >= 0),
    age_max         INT,
    price           INT NOT NULL CHECK (price >= 0),
    status          VARCHAR(10) NOT NULL CHECK (status IN ('draft', 'published')),
    epub_file_name  VARCHAR(255) NOT NULL,
    epub_size_bytes BIGINT NOT NULL,
    epub_content    BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    CHECK ((age_type = 'range' AND age_max IS NOT NULL AND age_max > age_min)
        OR (age_type = 'plus' AND age_max IS NULL))
);
CREATE INDEX books_showcase_idx ON books (created_at DESC) WHERE status = 'published' AND deleted_at IS NULL;

CREATE TABLE book_files (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id    UUID NOT NULL UNIQUE REFERENCES books (id),
    mime_type  VARCHAR(20) NOT NULL CHECK (mime_type IN ('image/jpeg', 'image/png')),
    file_name  VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    content    BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscription_plans (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    price      INT NOT NULL CHECK (price > 0),
    benefits   TEXT NOT NULL,
    duration   VARCHAR(20) NOT NULL DEFAULT '1 month',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id),
    plan_id         UUID NOT NULL REFERENCES subscription_plans (id),
    renewed_from_id UUID REFERENCES subscriptions (id),
    price_paid      INT NOT NULL CHECK (price_paid >= 0),
    status          VARCHAR(10) NOT NULL CHECK (status IN ('active', 'cancelled', 'expired')),
    started_at      TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    cancelled_at    TIMESTAMPTZ
);
-- At most one active period per parent: the DB rejects a concurrent second subscribe.
CREATE UNIQUE INDEX subscriptions_one_active_per_user ON subscriptions (user_id) WHERE status = 'active';
CREATE INDEX subscriptions_user_id_idx ON subscriptions (user_id, started_at DESC);
CREATE INDEX subscriptions_due_idx ON subscriptions (expires_at) WHERE status = 'active';

CREATE TABLE book_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id),
    book_id         UUID NOT NULL REFERENCES books (id),
    price           INT NOT NULL CHECK (price >= 0),
    status          VARCHAR(10) NOT NULL CHECK (status IN ('pending', 'paid', 'failed')),
    idempotency_key VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at         TIMESTAMPTZ,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX book_orders_book_id_idx ON book_orders (book_id);
CREATE INDEX book_orders_history_idx ON book_orders (user_id, created_at DESC);

CREATE TABLE subscription_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id),
    plan_id         UUID NOT NULL REFERENCES subscription_plans (id),
    subscription_id UUID REFERENCES subscriptions (id),
    order_type      VARCHAR(10) NOT NULL CHECK (order_type IN ('new', 'renewal')),
    price           INT NOT NULL CHECK (price >= 0),
    status          VARCHAR(10) NOT NULL CHECK (status IN ('pending', 'paid', 'failed')),
    idempotency_key VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at         TIMESTAMPTZ,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX subscription_orders_plan_id_idx ON subscription_orders (plan_id);
CREATE INDEX subscription_orders_subscription_id_idx ON subscription_orders (subscription_id);

CREATE TABLE purchases (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users (id),
    book_id       UUID NOT NULL REFERENCES books (id),
    book_order_id UUID NOT NULL UNIQUE REFERENCES book_orders (id),
    price_paid    INT NOT NULL CHECK (price_paid >= 0),
    purchased_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, book_id)
);
CREATE INDEX purchases_book_id_idx ON purchases (book_id);
