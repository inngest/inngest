SELECT * FROM runs r FULL JOIN (SELECT id AS run_id FROM events) e USING (run_id)
