--[[

Output:
    0: Success
   -1: No capacity left, not leased
   -2: Partition item not found
   -3: Partition item already leased

]]

local keyPartitionMap           = KEYS[1] -- key storing all partitions
local keyGlobalPartitionPtr     = KEYS[2] -- global top-level partitioned queue
local keyAccountPartitionPtr    = KEYS[3] -- account-level partitioned queue
local keyPartitionConcurrency   = KEYS[4] -- in progress queue for partition
local keyAccountConcurrency     = KEYS[5] -- in progress queue for account
local keyFnMeta                 = KEYS[6]


local partitionID             = ARGV[1]
local leaseID                 = ARGV[2]
local currentTime             = tonumber(ARGV[3]) -- in ms, to check lease validation
local leaseTime               = tonumber(ARGV[4]) -- in seconds, as partition score
local partitionConcurrency    = tonumber(ARGV[5]) -- concurrency limit for this partition
local accountConcurrency      = tonumber(ARGV[6]) -- concurrency limit for the acct. 
local noCapacityScore         = tonumber(ARGV[7]) -- score if concurrency is hit

-- $include(check_concurrency.lua)
-- $include(get_partition_item.lua)
-- $include(get_fn_meta.lua)
-- $include(decode_ulid_time.lua)
-- $include(update_pointer_score.lua)

local existing = get_partition_item(keyPartitionMap, partitionID)
if existing == nil or existing == false then
    return { -2 }
end

-- Check for an existing lease.
if existing.leaseID ~= nil and existing.leaseID ~= cjson.null and decode_ulid_time(existing.leaseID) > currentTime then
    return { -3 }
end

-- Check whether the partition is currently paused.
-- We only need to do this if the queue item is for a function.
if existing.wid ~= nil and existing.wid ~= cjson.null then
    local fnMeta = get_fn_meta(keyFnMeta)
    if fnMeta ~= nil and fnMeta.off then
        return {  -4 }
    end
end

local existingTime = existing.last

local capacity = partitionConcurrency -- initialize as the default concurrency limit
if partitionConcurrency > 0 and #keyPartitionConcurrency > 0 then
    -- Check that there's capacity for this partition, based off of partition-level
    -- concurrency keys.
    capacity = check_concurrency(currentTime, keyPartitionConcurrency, partitionConcurrency)
    if capacity <= 0 then
        requeue_partition(keyGlobalPartitionPtr, keyPartitionMap, existing, rartitionID, noCapacityScore, currentTime)
        requeue_partition(keyAccountPartitionPtr, keyPartitionMap, existing, rartitionID, noCapacityScore, currentTime)
        return { -1 }
    end
end

if accountConcurrency > 0 and #keyAccountConcurrency > 0 then
    -- Check that there's capacity for this partition, based off of partition-level
    -- concurrency keys.
    local acctCap = check_concurrency(currentTime, keyAccountConcurrency, accountConcurrency)
    if acctCap <= 0 then
        requeue_partition(keyGlobalPartitionPtr, keyPartitionMap, existing, rartitionID, noCapacityScore, currentTime)
        requeue_partition(keyAccountPartitionPtr, keyPartitionMap, existing, rartitionID, noCapacityScore, currentTime)
        return { -1 }
    end

    if acctCap <= capacity then
        capacity = acctCap
    end
end

existing.leaseID = leaseID
existing.at = leaseTime
existing.last = currentTime -- in ms.

-- Update item and index score
redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(existing))
redis.call("ZADD", keyGlobalPartitionPtr, leaseTime, partitionID) -- partition scored are in seconds.
redis.call("ZADD", keyAccountPartitionPtr, leaseTime, partitionID) -- partition scored are in seconds.
-- TODO Do we need to update the global account ZSET?

return { existingTime, capacity }
