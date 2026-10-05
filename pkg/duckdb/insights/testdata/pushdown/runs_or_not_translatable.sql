-- An OR with a branch that can't be translated (a function of a column), ILIKE, and a LIKE with ESCAPE yield no predicate.
SELECT
  run_id
FROM
  runs
WHERE
  (status = 'Failed' OR lower(app_id) = 'x')
  AND app_id ILIKE 'b%'
  AND app_id LIKE 'b!%' ESCAPE '!'
