SELECT * EXCLUDE (m.run_id, "e"."span_id") FROM metadata m JOIN extended_trace_spans e ON m.run_id = e.run_id
