--[[

  Requeues a partition at a specific time.
  will take into account the new priority when weighted sampling items to
  work on.

  Return values:
  0 - Updated priority
  1 - Partition not found
  2 - Garbage collected (but backlog still exists): Partition pointers removed
  3 - Garbage collected (all metadata dropped): Partition metadata deleted

]]

local keyPartitionHash              = KEYS[1]
local keyGlobalPartitionPtr         = KEYS[2]
local keyGlobalAccountPointer       = KEYS[3] -- accounts:sorted - zset
local keyAccountPartitions          = KEYS[4] -- accounts:$accountID:partition:sorted - zset
local keyPartitionMeta              = KEYS[5]
local keyFnMeta                     = KEYS[6]           -- fnMeta:$id - hash
local keyPartitionReady             = KEYS[7]
local keyPartitionInProgress        = KEYS[8] -- We can only GC a partition if no running jobs occur.
local queueKey                      = KEYS[9]
local keyShadowPartitionSet         = KEYS[10]
local keyPartitionConcurrencyIndex  = KEYS[11]

local partitionID             = ARGV[1]
local atMS                    = tonumber(ARGV[2]) -- time in milliseconds
local forceAt                 = tonumber(ARGV[3])
local accountID               = ARGV[4]

local atS = math.floor(atMS / 1000) -- in seconds;  partitions are currently second granularity, but this should change.

-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

--
local existing = get_partition_item(keyPartitionHash, partitionID)
if existing == nil then
    return 1
end

-- Always reset lease ID so next caller can lease partition
existing.leaseID = nil

-- update partition with removed lease ID
redis.call("HSET", keyPartitionHash, partitionID, cjson.encode(existing))

-- Remove partition from "in progress" ZSET
redis.call("ZREM", keyPartitionConcurrencyIndex, partitionID)

-- If there are no items in the workflow queue, we can safely remove the
-- partition.
local readyQueueEmpty = tonumber(redis.call("ZCARD", keyPartitionReady)) == 0
local inProgressEmpty = tonumber(redis.call("ZCARD", keyPartitionInProgress)) == 0
if readyQueueEmpty and inProgressEmpty then
  redis.call("ZREM", keyGlobalPartitionPtr, partitionID)    -- Remove the partition from global index

  if account_is_set(keyAccountPartitions) then
    redis.call("ZREM", keyAccountPartitions, partitionID)    -- Remove the partition from account index

    -- If this was the last account partition, remove account from global queue of accounts
    local numAccountPartitions = tonumber(redis.call("ZCARD", keyAccountPartitions))
    if numAccountPartitions == 0 then
      redis.call("ZREM", keyGlobalAccountPointer, accountID)
    end
  end

  -- Only drop partition information if no more backlogs exist for the partition
  if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
    redis.call("HDEL", keyPartitionHash, partitionID)             -- Remove the item
    redis.call("DEL", keyPartitionMeta)                         -- Remove the partition meta (this is to clean up legacy data)

    -- Clean up function metadata (which supersedes partition metadata)
    if exists_without_ending(keyFnMeta, ":fnMeta:-") == true then
      redis.call("DEL", keyFnMeta)
    end

    -- garbage collected: complete gc
    return 3
  end

  -- garbage collected: only removed pointers
  return 2
end

-- Peek up the next available item from the queue
local items = redis.call("ZRANGE", keyPartitionReady, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1)

if #items > 0 and forceAt ~= 1 then
    local encoded = redis.call("HMGET", queueKey, unpack(items))
    for k, v in pairs(encoded) do
				-- when an old executor processes a default partition, it does not
				-- remove pointers from key queues. we need to skip nil items in here
				if v ~= nil and v ~= false then
					local item = cjson.decode(v)
					if (item.leaseID == nil or item.leaseID == cjson.null) and math.floor(item.at / 1000) < atS then
							atS = math.floor(item.at / 1000)
							break
					end
				end
    end
end

if forceAt == 1 then
	existing.forceAtMS = atMS
else
	existing.forceAtMS = 0
end


existing.at = atS
redis.call("HSET", keyPartitionHash, partitionID, cjson.encode(existing))
update_pointer_score_to(partitionID, keyGlobalPartitionPtr, atS)
update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountID, atS)

return 0
