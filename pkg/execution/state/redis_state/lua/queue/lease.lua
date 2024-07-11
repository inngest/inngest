--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item already leased

  3: No function capacity
  4: No account capacity
  5: No custom capacity 1
  6: No custom capacity 2

  7: Rate limited via throttling;  no capacity.

]]

local queueKey               = KEYS[1]
local queueIndexKey          = KEYS[2]
local partitionKey           = KEYS[3]
-- We push our queue item ID into each concurrency queue
local accountConcurrencyKey  = KEYS[4] -- Account concurrency level
local functionConcurrencyKey = KEYS[5] -- When leasing an item we need to place the lease into this key.
local customConcurrencyKeyA  = KEYS[6] -- Optional for eg. for concurrency amongst steps
local customConcurrencyKeyB  = KEYS[7] -- Optional for eg. for concurrency amongst steps
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer     = KEYS[8]
local globalPointerKey       = KEYS[9]
local globalAccountKey       = KEYS[10]
local accountPointerKey      = KEYS[11]
local shardPointerKey        = KEYS[12]
local throttleKey            = KEYS[13] -- key used for throttling function run starts.

local queueID                = ARGV[1]
local newLeaseKey            = ARGV[2]
local currentTime            = tonumber(ARGV[3]) -- in ms
-- We check concurrency limits when leasing queue items: an account-level concurrency limit,
-- and a custom key.  The custom key is option.  It's used to add concurrency limits to individual
-- steps across differing functions
local accountConcurrency     = tonumber(ARGV[4])
local partitionConcurrency   = tonumber(ARGV[5])
local customConcurrencyA     = tonumber(ARGV[6])
local customConcurrencyB     = tonumber(ARGV[7])
local partitionName          = ARGV[8] -- Same as fn queue name/workflow ID
local accountId              = ARGV[9]

-- Use our custom Go preprocessor to inject the file from ./includes/
-- $include(decode_ulid_time.lua)
-- $include(check_concurrency.lua)
-- $include(get_queue_item.lua)
-- $include(set_item_peek_time.lua)
-- $include(update_pointer_score.lua)
-- $include(has_shard_key.lua)
-- $include(gcra.lua)

-- first, get the queue item.  we must do this and bail early if the queue item
-- was not found.
local item                   = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)
-- check if the item is leased.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
    -- This is already leased;  don't let this requester lease the item.
    return 2
end

-- Track the earliest time this job was attempted in the queue.
item = set_item_peek_time(queueKey, queueID, item, currentTime)

-- Track throttling/rate limiting IF the queue item has throttling info set.  This allows
-- us to target specific queue items with rate limiting individually.
--
-- We handle this before concurrency as it's typically not used, and it's faster to handle than concurrency,
-- with o(1) operations vs o(log(n)).
if item.data ~= nil and item.data.throttle ~= nil then
	local throttleResult = gcra(throttleKey, currentTime, item.data.throttle.p * 1000, item.data.throttle.l, item.data.throttle.b)
	if throttleResult == false then
		return 7
	end
end

-- Check the concurrency limits for the account and custom key;  partition keys are checked when
-- leasing the partition and do not need to be checked again (only one worker can run a partition at
-- once, and the capacity is kept in memory after leasing a partition)
if partitionConcurrency > 0 then
    if check_concurrency(currentTime, functionConcurrencyKey, partitionConcurrency) <= 0 then
        return 3
    end
end
if accountConcurrency > 0 then
    if check_concurrency(currentTime, accountConcurrencyKey, accountConcurrency) <= 0 then
        return 4
    end
end
if customConcurrencyA > 0 then
    if check_concurrency(currentTime, customConcurrencyKeyA, customConcurrencyA) <= 0 then
        return 5
    end
end
if customConcurrencyB > 0 then
    if check_concurrency(currentTime, customConcurrencyKeyB, customConcurrencyB) <= 0 then
        return 6
    end
end

-- Update the item's lease key.
item.leaseID = newLeaseKey
redis.call("HSET", queueKey, queueID, cjson.encode(item))

-- Add the item to all concurrency keys
redis.call("ZADD", functionConcurrencyKey, nextTime, item.id)

-- Remove the item from our sorted index, as this is no longer on the queue; it's in-progress
-- and store din functionConcurrencyKey.
redis.call("ZREM", queueIndexKey, item.id)

-- Update the fn's score in the global pointer queue to the next job, if available.
local score = get_fn_partition_score(queueIndexKey)
update_pointer_score_to(partitionName, globalPointerKey, score)
-- Also update account-level partitions
update_pointer_score_to(partitionName, accountPointerKey, score)
-- Also updated global accounts
-- TODO Is this correct?
update_pointer_score_to(accountId, globalAccountKey, score)
-- And the same for any shards, as long as the shard name exists.
if has_shard_key(shardPointerKey) then
    update_pointer_score_to(partitionName, shardPointerKey, score)
end

-- NOTE: We check if concurrency > 0 here because this disables concurrency.  AccountID
-- and custom concurrency items may not be set, but the keys need to be set for clustered
-- mode.
if accountConcurrency > 0 then
    redis.call("ZADD", accountConcurrencyKey, nextTime, item.id)
end
if customConcurrencyA > 0 then
    redis.call("ZADD", customConcurrencyKeyA, nextTime, item.id)
end
if customConcurrencyB > 0 then
    redis.call("ZADD", customConcurrencyKeyB, nextTime, item.id)
end

-- Get the earliest item in the partition/fn concurrency set - all items that have
-- been leased and are in progress.  If the current lease is the only item in the set, we'll
-- get the current lease.  Otherwise, we might get a lost job or a previously lost job.
local inProgressScores = redis.call("ZRANGE", functionConcurrencyKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1,
    "WITHSCORES")
if inProgressScores ~= false then
    local earliestLease = tonumber(inProgressScores[2])
    -- Add the earliest time to the pointer queue for in-progress, allowing us to scavenge
    -- lost jobs easily.
    redis.call("ZADD", concurrencyPointer, earliestLease, partitionName)
end

return 0
