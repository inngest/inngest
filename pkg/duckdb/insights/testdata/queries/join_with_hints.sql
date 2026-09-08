SELECT runs.run_id, extended_trace_spans.attributes ->> '_inngest.function.slug' FROM runs JOIN extended_trace_spans ON runs.run_id = extended_trace_spans.run_id
