  --[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseEventKey = KEYS[1]
local pauseInvokeKey = KEYS[2]
local keyPauseAddIdx = KEYS[3]
local keyPauseExpIdx = KEYS[4]

local pauseID       = ARGV[1]
local invokeCorrelationId = ARGV[2]


redis.call("HDEL", pauseEventKey, pauseID)

if invokeCorrelationId ~= false and invokeCorrelationId ~= "" and invokeCorrelationId ~= nil then
  redis.call("HDEL", pauseInvokeKey, invokeCorrelationId)
end

-- Add an index of when the pause was added.
redis.call("ZREM", keyPauseAddIdx, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZREM", keyPauseExpIdx, pauseID)


return 0
