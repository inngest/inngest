SELECT * FROM runs r SEMI JOIN extended_trace_spans e ON r.run_id = e.run_id JOIN metadata m USING (run_id)
