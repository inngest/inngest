-- Remove duplicate function entries, keeping the most recently created one per id.
-- This is necessary because the functions table was created without a PRIMARY KEY,
-- allowing duplicates to accumulate during sync operations.
DELETE FROM functions
WHERE ctid NOT IN (
    SELECT MAX(ctid)
    FROM functions
    GROUP BY id
);

-- Now that duplicates are removed, add the PRIMARY KEY constraint.
ALTER TABLE functions ADD PRIMARY KEY (id);
