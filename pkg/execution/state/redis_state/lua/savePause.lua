-- [[
--
-- Output:
--   0: Successfully saved pause
--   1: Pause already exists
-- ]]

local pauseKey    = KEYS[1]
local pauseEvtKey = KEYS[2]
local pauseInvokeKey = KEYS[3]
local keyPauseAddIdx = KEYS[4]
local keyPauseExpIdx = KEYS[5]
local keyRunPauses   = KEYS[6]

local pause          = ARGV[1]
local pauseID        = ARGV[2]
local event          = ARGV[3]
local invokeCorrelationID = ARGV[4]
local extendedExpiry = tonumber(ARGV[5])
local nowUnixSeconds = tonumber(ARGV[6])


if redis.call("SETNX", pauseKey, pause) == 0 then
	return 1
end

redis.call("EXPIRE", pauseKey, extendedExpiry)

-- Add an index of when the pause was added.
redis.call("ZADD", keyPauseAddIdx, nowUnixSeconds, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZADD", keyPauseExpIdx, nowUnixSeconds+extendedExpiry, pauseID)

-- SADD to store the pause for this run
redis.call("SADD", keyRunPauses, pauseID)

if event ~= false and event ~= "" and event ~= nil then
	redis.call("HSET", pauseEvtKey, pauseID, pause)
end

if invokeCorrelationID ~= false and invokeCorrelationID ~= "" and invokeCorrelationID ~= nil then
	redis.call("HSETNX", pauseInvokeKey, invokeCorrelationID, pauseID)
end

return 0
