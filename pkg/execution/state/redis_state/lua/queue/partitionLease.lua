--[[

Output:
    0: Success
   -1: No account capacity left, not leased
   -2: No fn capacity left, not leased
   -3: Partition item not found
   -4: Partition item already leased

]]

local keyPartitionMap         = KEYS[1] -- key storing all partitions
local keyGlobalPartitionPtr   = KEYS[2] -- global top-level partitioned queue
local keyGlobalAccountPointer = KEYS[3] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[4] -- accounts:$accountID:partition:sorted - zset

local partitionID             = ARGV[1]
local leaseID                 = ARGV[2]
local currentTime             = tonumber(ARGV[3]) -- in ms, to check lease validation
local leaseTime               = tonumber(ARGV[4]) -- in seconds, as partition score
local accountID               = ARGV[5]

-- $include(check_concurrency.lua)
-- $include(get_partition_item.lua)
-- $include(decode_ulid_time.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

local existing = get_partition_item(keyPartitionMap, partitionID)
if existing == nil or existing == false then
    return { -3 }
end

-- Check for an existing lease.
if existing.leaseID ~= nil and existing.leaseID ~= cjson.null and decode_ulid_time(existing.leaseID) > currentTime then
    return { -4 }
end

local existingTime = existing.last -- store a ref to the last time we successfully checked this partition

existing.leaseID = leaseID
existing.at = leaseTime
existing.last = currentTime -- in ms.

-- Update item and index score
redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(existing))
update_pointer_score_to(partitionID, keyGlobalPartitionPtr, leaseTime)
update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountID, leaseTime)

return { existingTime }
