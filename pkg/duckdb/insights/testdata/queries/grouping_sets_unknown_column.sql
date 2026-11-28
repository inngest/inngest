SELECT app_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (nonexistent_evil_column))
