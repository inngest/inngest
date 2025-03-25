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
local keyPartitionA           	= KEYS[9]           -- queue:sorted:$workflowID - zset
local keyPartitionB           	= KEYS[10]           -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC           	= KEYS[11]          -- e.g. sorted:c|t:$workflowID - zset
local keyItemIndexA           	= KEYS[12]          -- custom item index 1
local keyItemIndexB           	= KEYS[13]          -- custom item index 2

local queueItem           		= ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             		= ARGV[2]           -- id
local queueScore          		= tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       		= tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local nowMS               		= tonumber(ARGV[5]) -- now in ms
local fnMetadata          		= ARGV[6]          -- function meta: {paused}
local partitionItemA      		= ARGV[7]
local partitionItemB      		= ARGV[8]
local partitionItemC      		= ARGV[9]
local partitionIdA        		= ARGV[10]
local partitionIdB        		= ARGV[11]
local partitionIdC        		= ARGV[12]
local partitionTypeA        	= tonumber(ARGV[13])
local partitionTypeB        	= tonumber(ARGV[14])
local partitionTypeC        	= tonumber(ARGV[15])
local accountId           		= ARGV[16]
local guaranteedCapacity      = ARGV[17]
local guaranteedCapacityKey   = ARGV[18]

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

enqueue_to_partition(keyPartitionA, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionB, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionC, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)

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
