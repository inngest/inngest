
--[[

Output:
  0: Successfully cancelled
  1: Error setting status
]]

local metadataKey = KEYS[1]
local status      = tonumber(ARGV[1])

redis.call("HSET", metadataKey, "status", status)

return 0;
