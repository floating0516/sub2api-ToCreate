-- Link managed recharge inventory orders to externally paid payment orders.
-- Legacy rows keep payment_order_id NULL and retain their original balance-payment behavior.

ALTER TABLE managed_recharge_orders
    ADD COLUMN IF NOT EXISTS payment_order_id BIGINT REFERENCES payment_orders(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_managed_recharge_orders_payment_order
    ON managed_recharge_orders(payment_order_id)
    WHERE payment_order_id IS NOT NULL;
