-- runs joins the metadata rollup too, so metadata's predicates can't filter the shared delta.
SELECT
  r.run_id,
  m.metadata
FROM
  runs r
  JOIN metadata m ON m.run_id = r.run_id
WHERE
  m.span_id = 'span-1'
  AND r.status = 'Completed'
