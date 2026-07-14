UPDATE subscription_addon_products AS product
SET price = catalog.price,
    updated_at = NOW()
FROM (
    VALUES
        ('addon-usd-10', 1.99::NUMERIC),
        ('addon-usd-30', 5.49::NUMERIC),
        ('addon-usd-50', 8.49::NUMERIC),
        ('addon-usd-100', 14.99::NUMERIC),
        ('addon-usd-200', 27.99::NUMERIC)
) AS catalog(sku, price)
WHERE product.sku = catalog.sku;
