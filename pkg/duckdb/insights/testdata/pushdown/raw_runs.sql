-- Raw mode: physical columns directly, tenants included.
-- mode: raw
SELECT
  run_id
FROM
  runs
WHERE
  account_id = '00000000-0000-4000-a000-000000000000'
  AND status IN ('Completed', 'Failed')
  AND function_slug = 'my-fn'
