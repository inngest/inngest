--[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseKey      = KEYS[1]
local pauseEventKey = KEYS[2]
local pauseInvokeKey = KEYS[3]
local pauseSignalKey = KEYS[4]
local keyPauseAddIdx = KEYS[5]
local keyPauseExpIdx = KEYS[6]
local keyRunPauses   = KEYS[7]
local keyPausesIdx   = KEYS[8]
local keyPausesBlockIdx   = KEYS[9]

local pauseID       = ARGV[1]
local invokeCorrelationId = ARGV[2]
local signalCorrelationId = ARGV[3]
local blockIdxValue = ARGV[4]

redis.call("HDEL", pauseEventKey, pauseID)
redis.call("DEL", pauseKey)
-- SREM to remove the pause for this run
redis.call("SREM", keyRunPauses, pauseID)

-- Clean up global index
redis.call("SREM", keyPausesIdx, pauseID)

if invokeCorrelationId ~= false and invokeCorrelationId ~= "" and invokeCorrelationId ~= nil then
  redis.call("HDEL", pauseInvokeKey, invokeCorrelationId)
end

if signalCorrelationId ~= false and signalCorrelationId ~= "" and signalCorrelationId ~= nil then
  -- Ensure we only remove the signal if it belongs to this pause
  if redis.call("HGET", pauseSignalKey, signalCorrelationId) == pauseID then
    redis.call("HDEL", pauseSignalKey, signalCorrelationId)
  end
end

-- Add an index of when the pause was added.
redis.call("ZREM", keyPauseAddIdx, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZREM", keyPauseExpIdx, pauseID)


-- Add an index to block ID to be able to get pauses from block store by ID
if blockIdxValue ~= "" then
  redis.call("SET", keyPausesBlockIdx, blockIdxValue)
else
  redis.call("DEL", keyPausesBlockIdx)
end


return 0
