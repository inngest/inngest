SELECT run_id COLLATE nocase AS r, runs.app_id COLLATE nocase, defer_parent_run_ids[1] COLLATE nocase AS p FROM runs WHERE run_id COLLATE nocase IS NOT NULL
