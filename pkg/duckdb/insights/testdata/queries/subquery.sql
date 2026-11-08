SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM runs)
