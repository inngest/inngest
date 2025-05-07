--[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseKey      = KEYS[1]
local pauseEventKey = KEYS[2]
local pauseInvokeKey = KEYS[3]
local keyPauseAddIdx = KEYS[4]
local keyPauseExpIdx = KEYS[5]
local keyRunPauses   = KEYS[6]
local keyPausesIdx   = KEYS[7]

local pauseID       = ARGV[1]
local invokeCorrelationId = ARGV[2]

redis.call("HDEL", pauseEventKey, pauseID)
redis.call("DEL", pauseKey)
-- SREM to remove the pause for this run
redis.call("SREM", keyRunPauses, pauseID)

-- Clean up global index
redis.call("SREM", keyPausesIdx, pauseID)

if invokeCorrelationId ~= false and invokeCorrelationId ~= "" and invokeCorrelationId ~= nil then
  redis.call("HDEL", pauseInvokeKey, invokeCorrelationId)
end

-- Add an index of when the pause was added.
redis.call("ZREM", keyPauseAddIdx, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZREM", keyPauseExpIdx, pauseID)


return 0
