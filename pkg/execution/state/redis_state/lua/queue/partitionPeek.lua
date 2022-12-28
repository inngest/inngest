--[[

Peek returns partition items from the queue in order via their index.

]]

local partitionIndex = KEYS[1]
local partitionKey   = KEYS[2]

local peekUntil  = tonumber(ARGV[1])
local limit      = tonumber(ARGV[2])

-- local allowList  = cjson.decode(ARGV[3])
-- if allowList ~= nil and allowList ~= cjson.null then
-- 	-- If we're passing an allowlist, get all scores for each allowlist item
-- 	-- and see if the score for each item is less than the specified time.
-- 	local all = redis.call("ZMSCORE", partitionIndex, unpack(allowList))
-- 	local items = {}
-- 	for i=1, #all do
-- 		if all[i] <= allowList then
-- 			table.insert(items, #allowList[i])
-- 		end
-- 	end
-- 
-- 	return redis.call("HMGET", partitionKey, unpack(items))
-- end

local items = redis.call("ZRANGE", partitionIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
if #items == 0 then
	return {}
end

return redis.call("HMGET", partitionKey, unpack(items))
