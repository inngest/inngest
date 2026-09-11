SELECT app_id, function_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (function_id))
