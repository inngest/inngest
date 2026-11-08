SELECT run_id, SUM(1) OVER w FROM runs WINDOW w AS (PARTITION BY app_id ORDER BY run_id)
