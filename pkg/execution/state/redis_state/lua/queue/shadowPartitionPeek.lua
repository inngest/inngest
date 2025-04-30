--[[

shadowPartitionPeek returns backlogs from the shadow partition in order

Returns a tuple of
- the total number of backlogs up until peekUntil
- a list of backlog values potentially including nil values for dangling pointers
- a list of backlog IDs which can be correlated to missing values for cleanup

]]

local keyShadowPartitionSet = KEYS[1]
local keyBacklogMeta        = KEYS[2]

local peekUntilMS  = tonumber(ARGV[1])
local limit        = tonumber(ARGV[2])

local peekUntil    = math.ceil(peekUntilMS / 1000)

local totalCount = redis.call("ZCOUNT", keyShadowPartitionSet, "-inf", peekUntil)

local offset = 0

local backlogIDs = redis.call("ZRANGE", keyShadowPartitionSet, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #backlogIDs == 0 then
	return {}
end

local potentiallyMissingBacklogs = redis.call("HMGET", keyBacklogMeta, unpack(backlogIDs))

return {totalCount, potentiallyMissingBacklogs, backlogIDs}
