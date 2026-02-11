-- Add PRIMARY KEY to the functions table.
--
-- The original schema (000001) omitted the PRIMARY KEY on functions.id due to
-- a DuckDB limitation (https://github.com/duckdb/duckdb/issues/1631) that no
-- longer applies. Without a PRIMARY KEY, duplicate rows can accumulate during
-- sync operations, eventually causing "mismatched param and argument count"
-- errors when DeleteFunctionsByIDs is called.
--
-- Step 1: Remove duplicates, keeping the row with the latest created_at.
-- For rows with the same created_at, keep the one with the higher ctid
-- (physical row pointer) as a tiebreaker.
DELETE FROM functions f1
USING functions f2
WHERE f1.id = f2.id
  AND (f1.created_at < f2.created_at
       OR (f1.created_at = f2.created_at AND f1.ctid < f2.ctid));

-- Step 2: Add the PRIMARY KEY constraint now that duplicates are removed.
ALTER TABLE functions ADD PRIMARY KEY (id);
