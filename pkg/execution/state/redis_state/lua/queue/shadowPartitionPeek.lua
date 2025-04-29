--[[

shadowPartitionPeek returns backlogs from the shadow partition in order

]]

local keyShadowPartitionSet = KEYS[1]
local keyBacklogMeta        = KEYS[2]

local peekUntilMS  = tonumber(ARGV[1])
local limit        = tonumber(ARGV[2])

local peekUntil    = math.ceil(peekUntilMS / 1000)

local offset = 0

local backlogIDs = redis.call("ZRANGE", keyShadowPartitionSet, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #backlogIDs == 0 then
	return {}
end

local potentiallyMissingBacklogs = redis.call("HMGET", keyBacklogMeta, unpack(backlogIDs))

return {potentiallyMissingBacklogs, backlogIDs}
