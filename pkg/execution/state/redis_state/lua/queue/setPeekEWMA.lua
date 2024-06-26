--[[

sets the new value to the EWMA list
old values will be discarded as the size exceeds the limit

]]

local ewmaKey = KEYS[1]

local newValue = tonumber(ARGV[1])
local maxSize  = tonumber(ARGV[2])

-- the recent ones should go to the front
redis.call("LPUSH", ewmaKey, newValue)

local len = redis.call("LLEN", ewmaKey)
-- take out the oldest one when exceeded
if len > maxSize then
  redis.call("RPOP", ewmaKey)
end

-- expire or override the expiration of the key after 60s
-- this will make sure that functions that don't have a lot of load
-- won't take up a lot of key space
redis.call("EXPIRE", ewmaKey, 60)

return 0
