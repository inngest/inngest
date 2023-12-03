local pauseKey    = KEYS[1]
local stepKey     = KEYS[2]
local pauseEvtKey = KEYS[3]

local pause          = ARGV[1]
local pauseID        = ARGV[2]
local event          = ARGV[3] 
local newExpiry      = tonumber(ARGV[4])
local extendedExpiry = tonumber(ARGV[5])


if redis.call("SETNX", pauseKey, pause) == 0 then
	return 1
end

redis.call("EXPIRE", pauseKey, extendedExpiry)
redis.call("SETEX", stepKey, newExpiry, pauseID)

if event ~= false and event ~= "" and event ~= nil then
	redis.call("HSET", pauseEvtKey, pauseID, pause)
end

return 0
