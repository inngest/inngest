SELECT * FROM (WITH runs AS (SELECT 1 AS x) SELECT * FROM runs) a, runs
