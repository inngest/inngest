--[[

Peek returns items from the queue that are unleased.

]]

local queueIndex = KEYS[1]
local peekUntil  = ARGS[1]
local limit  = tonumber(ARGS[1])

local items = redis.call("ZRANGE", queueIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
return redis.call("MGET", unpack(items))
