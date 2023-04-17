
--[[

Output:
  0: Successfully cancelled
  1: Error setting status
]]

local metadataKey = KEYS[1]
local historyKey  = KEYS[2]

local status      = tonumber(ARGV[1])
local historyLog  = ARGV[2]
local logTime     = tonumber(ARGV[3])

redis.call("HSET", metadataKey, "status", status)
redis.call("ZADD", historyKey, logTime, historyLog)

return 0;
