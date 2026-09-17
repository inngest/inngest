SELECT run_id FROM runs WHERE queued_at > now() - INTERVAL 1 MINUTE
