--[[

Updates a debounce to use new data.

Return values:
- 0 (int): OK
- -1: Debounce is already in progress, as the queue item is leased.

]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key
-- We need queue details to check if the debounce job is in progress (leased).  If so, we fail
-- and create a new debounce job.
local keyQueueHash = KEYS[3]

local debounceID  = ARGV[1] 
local debounce    = ARGV[2]
local ttl         = tonumber(ARGV[3])
local queueJobID  = ARGV[4]
local currentTime = tonumber(ARGV[5]) -- in ms


-- copied from get_queue_item.lua
local function get_queue_item(queueKey, queueID)
	local fetched = redis.call("HGET", queueKey, queueID)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end

-- Check that the queue item is not leased (ie. this debounce is not in progress)
local item = get_queue_item(keyQueueHash, queueJobID)
if item == nil then
	-- The queue item was not found.  Return a new debounce.
	return -1
end
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
	-- The debounce queue item is leased. 
	return -1
end

-- Set the fn -> debounce ID pointer
redis.call("SETEX", keyPtr, ttl, debounceID)
redis.call("HSET", keyDbc, debounceID, debounce)

-- TODO: This should also reschedule the job directly in an atomic transaction.

return 0


