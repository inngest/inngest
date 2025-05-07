--[[

Retrieve backlogged queue items for normalization purposes

]]

local backlogKey = KEYS[1]
local queueItemKey = KEYS[2]

local limit = ARGV[1]

local total = redis.call("ZCARD", backlogKey)

local ids = redis.call("ZRANGE", backlogKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, limit)

local items = redis.call("HMGET", queueItemKey, unpack(ids))

return cjson.encode({ total = total, ids = ids, items = items })
