WITH x AS (SELECT run_id FROM runs) SELECT run_id, (SELECT count(*) AS n FROM x) AS n FROM runs
