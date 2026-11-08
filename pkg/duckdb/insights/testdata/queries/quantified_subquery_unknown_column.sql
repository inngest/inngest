SELECT run_id FROM runs WHERE run_id = ANY (SELECT id FROM events WHERE bogus_evil_column = 1)
