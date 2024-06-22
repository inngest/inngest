--[[

sets the new value to the EWMA list
old values will be discarded as the size exceeds the limit

]]

local ewmaKey = KEYS[1]

local newValue = ARGV[1]
local maxSize  = ARGV[2]

redis.call("RPUSH", ewmaKey)

local len = redis.call("LLEN", ewmaKey)
if len > maxSize then
  redis.call("LPOP", ewmaKey)
end

return 0
