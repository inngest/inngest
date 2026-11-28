WITH x AS (SELECT run_id FROM runs) SELECT run_id FROM runs WHERE EXISTS (SELECT run_id FROM x WHERE x.run_id = runs.run_id)
