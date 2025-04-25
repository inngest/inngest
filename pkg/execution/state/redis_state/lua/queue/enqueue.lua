--[[

Enqueus an item within the queue.


--]]

local queueKey                	= KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         	= KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        	= KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer 	= KEYS[4]           -- accounts:sorted - zset
local keyAccountPartitions    	= KEYS[5]           -- accounts:$accountId:partition:sorted - zset
local idempotencyKey          	= KEYS[6]           -- seen:$key
local keyFnMetadata           	= KEYS[7]           -- fnMeta:$id - hash
local guaranteedCapacityMapKey	= KEYS[8]           -- shards - hmap of shards
local keyPartition           	  = KEYS[9]           -- queue:sorted:$workflowID - zset

-- Key queues v2
local keyBacklogSetA              = KEYS[10]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetB              = KEYS[11]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetC              = KEYS[12]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta              = KEYS[13]          -- backlogs - hash
local keyGlobalShadowPartitionSet = KEYS[14]          -- shadow:sorted
local keyShadowPartitionSet       = KEYS[15]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta      = KEYS[16]          -- shadows

local keyItemIndexA           	= KEYS[17]          -- custom item index 1
local keyItemIndexB           	= KEYS[18]          -- custom item index 2


local queueItem           		= ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             		= ARGV[2]           -- id
local queueScore          		= tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       		= tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local nowMS               		= tonumber(ARGV[5]) -- now in ms
local fnMetadata          		= ARGV[6]          -- function meta: {paused}
local partitionItem      		  = ARGV[7]
local partitionID        		  = ARGV[8]
local accountId           		= ARGV[9]
local guaranteedCapacity      = ARGV[10]
local guaranteedCapacityKey   = ARGV[11]

-- Key queues v2
local enqueueToBacklog				= tonumber(ARGV[12])
local shadowPartitionItem     = ARGV[13]
local backlogItemA            = ARGV[14]
local backlogItemB            = ARGV[15]
local backlogItemC            = ARGV[16]
local backlogIdA              = ARGV[17]
local backlogIdB              = ARGV[18]
local backlogIdC              = ARGV[19]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(get_partition_item.lua)
-- $include(enqueue_to_partition.lua)
-- $include(ends_with.lua)

-- Check idempotency exists
if redis.call("EXISTS", idempotencyKey) ~= 0 then
    return 1
end

-- Make these a hash to save on memory usage
if redis.call("HSETNX", queueKey, queueID, queueItem) == 0 then
    -- This already exists;  return an error.
    return 1
end

if enqueueToBacklog == 1 then
	-- the default function queue could be any of the three, usually the first but possibly the middle or last if a custom concurrency key is used

	enqueue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, partitionTime, nowMS)
	enqueue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, partitionTime, nowMS)
	enqueue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, partitionTime, nowMS)
else
  enqueue_to_partition(keyPartition, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
end

if exists_without_ending(keyFnMetadata, ":fnMeta:-") == true then
	-- note to future devs: if updating metadata, be sure you do not change the "off"
	-- (i.e. "paused") boolean in the function's metadata.
	redis.call("SET", keyFnMetadata, fnMetadata, "NX")
end

if guaranteedCapacityKey ~= "" then
	-- If no guaranteed capacity is defined, remove key from map
	if guaranteedCapacity ~= "" and guaranteedCapacity ~= "null" then
		-- If the account has guaranteed capacity, upsert the guaranteed capacity map.
		-- NOTE: We do not want to overwrite the account leases, so here
		-- we fetch the guaranteed capacity item, set the lease values in the passed in guaranteed capacity
		-- item, then write the updated value.
		local existingGuaranteedCapacity = redis.call("HGET", guaranteedCapacityMapKey, guaranteedCapacityKey)
		if existingGuaranteedCapacity ~= nil and existingGuaranteedCapacity ~= false then
			local updatedGuaranteedCapacity = cjson.decode(guaranteedCapacity)
			existingGuaranteedCapacity = cjson.decode(existingGuaranteedCapacity)
			updatedGuaranteedCapacity.leases = existingGuaranteedCapacity.leases
			guaranteedCapacity = cjson.encode(updatedGuaranteedCapacity)
		end
		redis.call("HSET", guaranteedCapacityMapKey, guaranteedCapacityKey, guaranteedCapacity)
	else
		-- Note: This code path is hit by every enqueue that does not have guaranteed capacity
		-- and might never have used guaranteed capacity in the first place. We can also remove this
		-- as long as we remove guaranteed capacity from the map manually whenever accounts lose access.
		redis.call("HDEL", guaranteedCapacityMapKey, guaranteedCapacityKey)
	end
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
