ALTER TABLE managed_recharge_orders
    ADD COLUMN IF NOT EXISTS fulfillment_mode VARCHAR(16) NOT NULL DEFAULT 'proxy';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'managed_recharge_orders_fulfillment_mode_check'
    ) THEN
        ALTER TABLE managed_recharge_orders
            ADD CONSTRAINT managed_recharge_orders_fulfillment_mode_check
            CHECK (fulfillment_mode IN ('proxy', 'external'));
    END IF;
END $$;
