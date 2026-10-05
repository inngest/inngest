-- A column equality is pushed, on the physical column its logical one maps to.
SELECT
  run_id
FROM
  runs
WHERE
  function_id = 'my-fn'
