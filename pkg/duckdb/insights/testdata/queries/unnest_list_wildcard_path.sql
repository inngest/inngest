SELECT UNNEST(list(inputs) -> '$[*][*].meta.sessions') FROM runs
