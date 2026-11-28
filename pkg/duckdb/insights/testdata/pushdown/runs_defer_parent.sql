-- defer_parent_function_id maps to the physical defer_parent_fn_slug.
SELECT
  run_id
FROM
  runs
WHERE
  defer_parent_function_id = 'parent-fn'
