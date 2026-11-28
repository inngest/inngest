SELECT now() AS t FROM runs UNION ALL SELECT now() FROM runs ORDER BY now() LIMIT 5 OFFSET (SELECT count(*) FROM events)
