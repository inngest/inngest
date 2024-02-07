--[[

Enqueus an item within the queue.


--]]

local queueKey            = KEYS[1]           -- queue:item - hash: { $itemID: $item }
local queueIndexKey       = KEYS[2]           -- queue:sorted:$workflowID - zset
local partitionKey        = KEYS[3]           -- partition:item - hash: { $workflowID: $partition }
local partitionCounterKey = KEYS[4]           -- partition:item:$workflowID - hash
local partitionIndexKey   = KEYS[5]           -- partition:sorted - zset
local shardIndexKey       = KEYS[6]           -- shard:$name:sorted - zset
local shardMapKey         = KEYS[7]           -- shards - hmap of shards
local idempotencyKey      = KEYS[8]           -- seen:$key
local keyItemIndexA       = KEYS[9]           -- custom item index 1
local keyItemIndexB       = KEYS[10]          -- custom item index 2

local queueItem           = ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in milliseconds
local workflowID          = ARGV[4]           -- $workflowID
local partitionItem       = ARGV[5]           -- {workflow, priority, leasedAt, etc}
local partitionTime       = tonumber(ARGV[6]) -- score for partition, lower bounded to now in seconds
local shard               = ARGV[7]
local shardName           = ARGV[8]

-- $include(get_partition_item.lua)

-- Check idempotency exists
if redis.call("EXISTS", idempotencyKey) ~= 0 then
    return 1
end

-- Make these a hash to save on memory usage
if redis.call("HSETNX", queueKey, queueID, queueItem) == 0 then
    -- This already exists;  return an error.
    return 1
end

-- We score the queue items separately, as we need to continually update the score
-- when adding leases.  Doing so means we can't ZADD to update sorted sets, as each
-- time the lease ID changes the data structure changes; zsets require static members
-- when updating scores.
redis.call("ZADD", queueIndexKey, queueScore, queueID)

-- We store partitions and their leases separately from the queue-partition ZSET
-- as we want a static member excluding eg. lease IDs.  This allows us to update
-- scores idempotently.
redis.call("HSETNX", partitionKey, workflowID, partitionItem)
redis.call("HSETNX", partitionCounterKey, "n", 0)    -- Atomic counter, currently leased (in progress) items.
redis.call("HINCRBY", partitionCounterKey, "len", 1) -- Atomic counter, length of enqueued items, set to 1 or increased.

-- If this is a sharded item, upsert the shard.
if shard ~= "" and shard ~= "null" then
    -- NOTE: We do not want to overwrite the shard leases, so here
    -- we fetch the shard item, set the lease values in the passed in shard
    -- item, then write the updated value.
    local existingShard = redis.call("HGET", shardMapKey, shardName)
    if existingShard ~= nil and existingShard ~= false then
	local updatedShard = cjson.decode(shard)
	existingShard = cjson.decode(existingShard)
        updatedShard.leases = existingShard.leases
        shard = cjson.encode(updatedShard)
    end
    redis.call("HSET", shardMapKey, shardName, shard)
end

-- Get the current score of the partition;  if queueScore < currentScore update the
-- partition's score so that we can work on this workflow when the earliest member
-- is available.
local currentScore = redis.call("ZSCORE", partitionIndexKey, workflowID)
if currentScore == false or tonumber(currentScore) > partitionTime then
    redis.call("ZADD", partitionIndexKey, partitionTime, workflowID)

    -- if this is sharded we have a shard partition to update.
    if shard ~= "" and shard ~= "null" then
        redis.call("ZADD", shardIndexKey, partitionTime, workflowID)
    end

    -- Get the partition item, so that we can keep the last lease score.
    local existing = get_partition_item(partitionKey, workflowID)
    if existing ~= nil then
        local decoded = cjson.decode(partitionItem)
        decoded.last = existing.last
        partitionItem = cjson.encode(decoded)
    end

    -- Set the partition item.  We must always do this so that we can
    -- update priorities on the fly.
    redis.call("HSET", partitionKey, workflowID, partitionItem)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

-- TODO: For the given workflow ID increase scheduled count, store a history item,
-- etc:  this can be atomic in the redis queue as it combines state + queue.

return 0
