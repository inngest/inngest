-- gets a decoded partition item
local function enqueue_get_partition_item(partitionKey, id)
	local fetched = redis.call("HGET", partitionKey, id)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end

local function enqueue_to_partition(keyPartitionSet, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
	if partitionID == "" then
		-- This is a blank partition, so don't even bother.  This allows us to pre-allocate
		-- 3 partitions per item, even if an item only needs a single partition.
		return
	end

	-- Push the queue item's ID to the given partition set.
	redis.call("ZADD", keyPartitionSet, queueScore, queueID)

	-- NOTE: For backwards compatibility, if a function has no concurrency or throttling keys it's
	--       partition set is "{q:v1}:queue:sorted:$workflowID", and the member stored in the global
	--       set of functions is *just* the workflow ID.
	--
	--       For new key-based queues, we actually store the entire redis key here.  Much better.
	--       
	--       Because of this discrepancy, we have to pass in a "partitionID" to this function so
	--       that we can properly do backcompat in the global queue of queues.

	-- Potentially update the queue of queues.  
	local currentScore = redis.call("ZSCORE", keyGlobalPointer, partitionID) 
	if currentScore == false or tonumber(currentScore) > partitionTime then
		-- In this case, we're enqueueing something earlier than we previously had in
		-- the current queue/partition.  To this effect, we need to:
		--   1. Update the queue of queues.
		--   2. Track some metadata in the current queue/partition item, because of things.

		-- Get the partition item, so that we can keep the last lease score.
		local existing = enqueue_get_partition_item(keyPartitionMap, partitionID)
		-- NOTE: There's a concept of "forcing" a partition not to be evaluated until a
		--       specific time.  We want to do this to reduce contention.  It makes sense.
		--       Trust me.
		--
		--       Because of this, we don't want to continually update the global order if
		--       we've forced a partition to have a delay.
		--
		--       Here, we do those checks.
		if nowMS > existing.forceAtMS then
			-- If the current time is before the force stuff, don't bother.  Here, we
			-- are guaranteed that we've already passed the force delay.
			--
			-- This is the case when there's no force delay or we've waited enough time.
			-- So, update the global index such that this partition is found, plz. Tyvm!!
			redis.call("ZADD", keyGlobalPointer, partitionTime, partitionID)
		end
	end

  -- Potentially update the account partition pointer
  local currentScore = redis.call("ZSCORE", keyAccountPartitions, partitionID)
  if currentScore == false or tonumber(currentScore) > partitionTime then
    local existing = enqueue_get_partition_item(keyPartitionMap, partitionID)
    if nowMS > existing.forceAtMS then
      redis.call("ZADD", keyAccountPartitions, partitionTime, partitionID)
    end
  end
end

-- requeue_to_partition is similar to enqueue, but always fetches the minimum score for a partition to
-- update global pointers instead of using the current queue item's score.
-- Requires: update_account_queues.lua which requires update_pointer_score.lua, ends_with.lua
local function requeue_to_partition(keyPartitionSet, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountID)
	if partitionID == "" then
		-- This is a blank partition, so don't even bother.  This allows us to pre-allocate
		-- 3 partitions per item, even if an item only needs a single partition.
		return
	end

	-- Push the queue item's ID to the given partition set.
	redis.call("ZADD", keyPartitionSet, queueScore, queueID)

	-- NOTE: For backwards compatibility, if a function has no concurrency or throttling keys it's
	--       partition set is "{q:v1}:queue:sorted:$workflowID", and the member stored in the global
	--       set of functions is *just* the workflow ID.
	--
	--       For new key-based queues, we actually store the entire redis key here.  Much better.
	--       
	--       Because of this discrepancy, we have to pass in a "partitionID" to this function so
	--       that we can properly do backcompat in the global queue of queues.
	redis.call("HSETNX", keyPartitionMap, partitionID, partitionItem) -- store the partition

	-- Get the minimum score for the queue.
	local minScores = redis.call("ZRANGE", keyPartitionSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
	local earliestScore = tonumber(minScores[2])

	-- Potentially update the queue of queues.  
	local currentScore = redis.call("ZSCORE", keyGlobalPointer, partitionID) 
	if currentScore == false or tonumber(currentScore) ~= earliestScore then
		-- In this case, we're enqueueing something earlier than we previously had in
		-- the current queue/partition.  To this effect, we need to:
		--   1. Update the queue of queues.
		--   2. Track some metadata in the current queue/partition item, because of things.

		-- Get the partition item, so that we can keep the last lease score.
		local existing = enqueue_get_partition_item(keyPartitionMap, partitionID)
		-- NOTE: There's a concept of "forcing" a partition not to be evaluated until a
		--       specific time.  We want to do this to reduce contention.  It makes sense.
		--       Trust me.
		--
		--       Because of this, we don't want to continually update the global order if
		--       we've forced a partition to have a delay.
		--
		--       Here, we do those checks.
		if nowMS > existing.forceAtMS then
			-- If the current time is before the force stuff, don't bother.  Here, we
			-- are guaranteed that we've already passed the force delay.
			--
			-- This is the case when there's no force delay or we've waited enough time.
			-- So, update the global index such that this partition is found, plz. Tyvm!!
			updateTo = earliestScore/1000

			update_pointer_score_to(partitionID, keyGlobalPointer, updateTo)
      update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountId, updateTo)
		end
	end
end
