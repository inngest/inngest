SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs LIMIT 5 OFFSET length(read_text('x'))
