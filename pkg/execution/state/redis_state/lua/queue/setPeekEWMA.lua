--[[

sets the new value to the EWMA list
old values will be discarded as the size exceeds the limit

]]

local ewmaKey = KEYS[1]

local newValue = ARGV[1]
local maxSize  = ARGV[2]

redis.call("RPUSH", ewmaKey)

local len = redis.call("LLEN", ewmaKey)
-- take out the oldest one when exceeded
if len > maxSize then
  redis.call("LPOP", ewmaKey)
end

-- expire or override the expiration of the key after 30s
-- this will make sure that functions that don't have a lot of load
-- won't take up a lot of key space
redis.call("EXPIRE", ewmaKey, 30)

return 0
