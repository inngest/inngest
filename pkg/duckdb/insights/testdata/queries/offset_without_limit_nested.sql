SELECT * FROM (SELECT run_id FROM runs OFFSET 5) s WHERE run_id IN (SELECT run_id FROM runs OFFSET 1)
