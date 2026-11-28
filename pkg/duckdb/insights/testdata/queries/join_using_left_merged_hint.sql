SELECT run_id FROM runs r LEFT JOIN (SELECT id AS run_id FROM events) e USING (run_id)
