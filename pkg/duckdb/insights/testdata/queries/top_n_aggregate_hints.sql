SELECT max(run_id, 3) AS ids, arg_max(function_id, queued_at, 2) AS fns, max(run_id) AS last_id FROM runs
