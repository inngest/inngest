-- Checks whether there's capacity in the given concurrency queue, given a limit and the
-- current time in milliseconds.
local function check_concurrency(now_ms, key, limit)
	local count = redis.call("ZCOUNT", key, tostring(now_ms), "+inf")
	return tonumber(limit) - tonumber(count)
end

local function requeue_partition(keyZset, keyPartitionMap, partition, partitionID, score, currentTime)
        -- Update that we attempted to lease this partition, even if there was no capacity.
        partition.last = currentTime -- in ms.
        redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(partition))
        -- There's no capacity available.  Increase the score for this partition so that
        -- it's not immediately re-scanned.
        redis.call("ZADD", keyZset, score, partitionID)
end
