SELECT run_id, SUM(1) OVER w FROM runs WINDOW w AS (PARTITION BY nonexistent_evil_column)
