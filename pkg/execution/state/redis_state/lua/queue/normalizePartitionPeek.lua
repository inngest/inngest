--[[

Returns backlogs from the normalization partition

]]

local partitionKey = KEYS[1]
local backlogKey = KEYS[2]

local limit = tonumber(ARGV[1])


local count = redis.call("ZCOUNT", partitionKey, "-inf", "+inf")

local backlogIDs = redis.call("ZRANGE", partitionKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, limit)
if #backlogIDs == 0 then
  return {}
end

local backlogs = redis.call("HMGET", backlogKey, unpack(backlogIDs))

return cjson.encode({ count = count, backlogs = backlogs, ids = backlogIDs})
