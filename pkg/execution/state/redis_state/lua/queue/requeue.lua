--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found

]]

local queueKey                = KEYS[1] -- queue:item - hash: { $itemID: $item }
local queueIndexKey           = KEYS[2] -- queue:sorted:$workflowID - zset
local partitionKey            = KEYS[3] -- partition:item:$workflowID - hash { n: $leased, len: $enqueued }
local partitionIndexKey       = KEYS[4] -- partition:sorted - zset
-- We push our queue item ID into each concurrency queue
local accountConcurrencyKey   = KEYS[5] -- Account concurrency level
local partitionConcurrencyKey = KEYS[6] -- When leasing an item we need to place the lease into this key.
local customConcurrencyKeyA   = KEYS[7] -- Optional for eg. for concurrency amongst steps
local customConcurrencyKeyB   = KEYS[8] -- Optional for eg. for concurrency amongst steps
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[9]
local shardPointerKey         = KEYS[10]
local keyItemIndexA           = KEYS[11]          -- custom item index 1
local keyItemIndexB           = KEYS[12]          -- custom item index 2

local queueItem               = ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID                 = ARGV[2]           -- id
local queueScore              = tonumber(ARGV[3]) -- vesting time, in ms
local functionID              = ARGV[4]           -- workflowID
local partitionItem           = ARGV[5]           -- {workflow, priority, leasedAt, etc}

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(has_shard_key.lua)

local item                    = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

if item.leaseID ~= nil and item.leaseID ~= cjson.null then
    -- Remove total number in progress if there's a lease.
    -- XXX: This is unused and is a rough indicator.  Use concurrency queues for
    -- an actual indicator.
    redis.call("HINCRBY", partitionKey, "n", -1)
end

redis.call("HSET", queueKey, queueID, queueItem)
-- Update the queue score
redis.call("ZADD", queueIndexKey, queueScore, queueID)

-- Remove this from all in-progress queues
redis.call("ZREM", partitionConcurrencyKey, item.id)
if accountConcurrencyKey ~= nil and accountConcurrencyKey ~= "" then
    redis.call("ZREM", accountConcurrencyKey, item.id)
end
if customConcurrencyKeyA ~= nil and customConcurrencyKeyA ~= "" then
    redis.call("ZREM", customConcurrencyKeyA, item.id)
end
if customConcurrencyKeyB ~= nil and customConcurrencyKeyB ~= "" then
    redis.call("ZREM", customConcurrencyKeyB, item.id)
end

-- Fetch partition index;  ensure this is the same as our lowest queue item score
local currentScore = redis.call("ZSCORE", partitionIndexKey, functionID)

-- Peek the earliest time from the queue index.  We need to know
-- the earliest time for the entire function set, as we may be
-- rescheduling the only time in the queue;  this is the only way
-- to update the partiton index.
local earliestTime = get_fn_partition_score(queueIndexKey)

-- earliest is a table containing {item, score}
if currentScore == false or tonumber(currentScore) ~= earliestTime or tonumber(currentScore) == nil then
    redis.call("ZADD", partitionIndexKey, earliestTime, functionID)
    -- And the same for any shards, as long as the shard name exists.
    if has_shard_key(shardPointerKey, -2) then
        update_pointer_score_to(functionID, shardPointerKey, earliestTime)
    end
end

-- Get the earliest item in the partition concurrency set.  We may be requeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
local concurrencyScores = redis.call("ZRANGE", partitionConcurrencyKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1,
    "WITHSCORES")
if concurrencyScores == false then
    redis.call("ZREM", concurrencyPointer, functionID)
else
    local earliestLease = tonumber(concurrencyScores[2])
    if earliestLease == nil then
        redis.call("ZREM", concurrencyPointer, functionID)
    else
        -- Ensure that we update the score with the earliest lease
        redis.call("ZADD", concurrencyPointer, earliestLease, functionID)
    end
end


-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
