SELECT s.k, [x FOR x IN runs.sessions.key IF x IS NOT NULL] AS ks FROM runs, LATERAL (SELECT sessions.key AS k) s
