WITH x AS (SELECT r.run_id, e.run_id, e.app_id FROM runs r JOIN extended_trace_spans e ON r.run_id = e.run_id) SELECT run_id_1, app_id FROM x
