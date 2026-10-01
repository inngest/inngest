WITH c AS (SELECT run_id AS getenv FROM runs) SELECT getenv OPERATOR(+) ('HOME' || '') AS leaked FROM c
