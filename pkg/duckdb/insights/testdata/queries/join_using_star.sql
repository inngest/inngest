SELECT * FROM metadata JOIN extended_trace_spans USING (run_id, span_id)
