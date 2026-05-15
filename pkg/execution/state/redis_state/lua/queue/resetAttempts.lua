--[[

Resets a job's internal attempt count to zero.


Return values:

- 0:  Successfully requeued
- -1: Queue item not found

]]
--

local keyQueueHash = KEYS[1] -- queue:item - hash
local jobID = ARGV[1] -- queue item ID

-- $include(get_queue_item.lua)

local item = get_queue_item(keyQueueHash, jobID)
if item == nil then
	return -1
end

-- Update the "at" time of the job
item.atts = 0
redis.call("HSET", keyQueueHash, jobID, cjson.encode(item))

return 0
