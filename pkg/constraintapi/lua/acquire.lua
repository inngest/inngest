--
-- Acquire
-- - checks constraint capacity given constraints and the current configuration
-- -
--

local keyLeasesHash = KEYS[1]

local requestDetails = ARGV[1]

-- TODO: Handle operation idempotency

-- TODO: Verify no far newer config was seen (reduce driftt)

-- TODO: Handle constraint idempotency (do we need to skip GCRA?)

-- TODO: Compute constraint capacity

-- TODO: If missing capacity, exit early (return limiting constraint and details)

-- TODO: Update constraint state with granted capacity

-- TODO: Store request details, granted capacity

-- TODO: Add lease to active leases for account (and account pointer to set to scavenger shard)

-- TODO: Set operation idempotency key

return 0
