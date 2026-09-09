-- Stop collapsing unlisted monitor models into __other__ / "其他模型".
-- Presentation now keeps every stored model name; empty models lists make the
-- admin allow-list match that behavior on existing deployments.
UPDATE channel_monitor_v2_config
SET
    platforms = (
        SELECT COALESCE(jsonb_agg(jsonb_set(elem, '{models}', '[]'::jsonb)), '[]'::jsonb)
        FROM jsonb_array_elements(platforms) AS elem
    ),
    version = version + 1,
    updated_at = NOW()
WHERE id = 1
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(platforms) AS elem
      WHERE jsonb_typeof(elem->'models') = 'array'
        AND jsonb_array_length(elem->'models') > 0
  );
