CREATE TABLE IF NOT EXISTS subscription_addon_products (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    quota_usd NUMERIC(20, 10) NOT NULL,
    price NUMERIC(20, 2) NOT NULL,
    original_price NUMERIC(20, 2) NULL,
    for_sale BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_addon_products_quota_positive CHECK (quota_usd > 0),
    CONSTRAINT subscription_addon_products_price_positive CHECK (price > 0),
    CONSTRAINT subscription_addon_products_original_price_nonnegative CHECK (original_price IS NULL OR original_price >= 0)
);

CREATE INDEX IF NOT EXISTS subscription_addon_products_for_sale_sort_idx
    ON subscription_addon_products (for_sale, sort_order, id);

ALTER TABLE subscription_addon_packs
    ADD COLUMN IF NOT EXISTS purchase_order_id BIGINT NULL REFERENCES payment_orders(id);

CREATE UNIQUE INDEX IF NOT EXISTS subscription_addon_packs_purchase_order_unique_idx
    ON subscription_addon_packs (purchase_order_id)
    WHERE purchase_order_id IS NOT NULL;

INSERT INTO subscription_addon_products (sku, name, quota_usd, price, sort_order)
VALUES
    ('addon-usd-10', '10 美元加油包', 10, 2.99, 10),
    ('addon-usd-30', '30 美元加油包', 30, 7.99, 20),
    ('addon-usd-50', '50 美元加油包', 50, 12.99, 30),
    ('addon-usd-100', '100 美元加油包', 100, 23.99, 40),
    ('addon-usd-200', '200 美元加油包', 200, 44.99, 50)
ON CONFLICT (sku) DO NOTHING;
