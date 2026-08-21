-- Managed recharge inventory and fulfillment orders.
-- Sensitive CDK/session values are stored as AES-GCM ciphertext only.

CREATE TABLE IF NOT EXISTS managed_recharge_products (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(64) NOT NULL UNIQUE,
    plan_type VARCHAR(32) NOT NULL UNIQUE CHECK (plan_type IN ('plus', 'pro')),
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price NUMERIC(18, 6) NOT NULL CHECK (price >= 0),
    active BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS managed_recharge_cdks (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES managed_recharge_products(id) ON DELETE RESTRICT,
    code_ciphertext TEXT NOT NULL,
    code_hash CHAR(64) NOT NULL UNIQUE,
    code_masked VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'available',
    expires_at TIMESTAMPTZ,
    reserved_order_id BIGINT,
    reserved_at TIMESTAMPTZ,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS managed_recharge_orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(48) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    product_id BIGINT NOT NULL REFERENCES managed_recharge_products(id) ON DELETE RESTRICT,
    cdk_id BIGINT REFERENCES managed_recharge_cdks(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(128) NOT NULL,
    price NUMERIC(18, 6) NOT NULL CHECK (price >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'validating',
    account_email VARCHAR(255) NOT NULL DEFAULT '',
    session_ciphertext TEXT NOT NULL DEFAULT '',
    upstream_task_id VARCHAR(128) NOT NULL DEFAULT '',
    upstream_status VARCHAR(64) NOT NULL DEFAULT '',
    upstream_failure_reason TEXT NOT NULL DEFAULT '',
    queue_position INTEGER NOT NULL DEFAULT 0,
    queue_total INTEGER NOT NULL DEFAULT 0,
    progress TEXT NOT NULL DEFAULT '',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    balance_before NUMERIC(18, 6),
    balance_after NUMERIC(18, 6),
    paid_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS managed_recharge_events (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES managed_recharge_orders(id) ON DELETE CASCADE,
    actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_managed_recharge_products_active_sort
    ON managed_recharge_products(active, sort_order, id);
CREATE INDEX IF NOT EXISTS idx_managed_recharge_cdks_product_status
    ON managed_recharge_cdks(product_id, status, expires_at, id);
CREATE INDEX IF NOT EXISTS idx_managed_recharge_cdks_reserved_order
    ON managed_recharge_cdks(reserved_order_id);
CREATE INDEX IF NOT EXISTS idx_managed_recharge_orders_user_created
    ON managed_recharge_orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_managed_recharge_orders_status_sync
    ON managed_recharge_orders(status, last_synced_at);
CREATE INDEX IF NOT EXISTS idx_managed_recharge_events_order_created
    ON managed_recharge_events(order_id, created_at DESC);
