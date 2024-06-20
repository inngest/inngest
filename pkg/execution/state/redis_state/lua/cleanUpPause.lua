--[[

After a pause has been consumed, this script runs to clean up the pause's
presence in any global (unsharded) keys.

Output:
  0: Successfully cleaned up

]]

local pauseStepKey  = KEYS[1]
local pauseEventKey = KEYS[2]
local pauseInvokeKey = KEYS[3]
local keyPauseAddIdx = KEYS[4]
local keyPauseExpIdx = KEYS[5]
local keyRunPauses   = KEYS[6]

local pauseID      = ARGV[1]
local invokeCorrelationId = ARGV[2]

if pauseEventKey ~= "" then
	-- Clean up regardless
	redis.call("HDEL", pauseEventKey, pauseID)
	-- Add an index of when the pause was added.
	redis.call("ZREM", keyPauseAddIdx, pauseID)
	-- Add an index of when the pause expires.  This lets us manually
	-- garbage collect expired pauses from the HSET below.
	redis.call("ZREM", keyPauseExpIdx, pauseID)
end

redis.call("DEL", pauseStepKey)
-- SREM to remove the pause for this run
redis.call("SREM", keyRunPauses, pauseID)

if pauseEventKey ~= "" then
	redis.call("HDEL", pauseEventKey, pauseID)
end

if invokeCorrelationId ~= false and invokeCorrelationId ~= "" and invokeCorrelationId ~= nil then
	redis.call("HDEL", pauseInvokeKey, invokeCorrelationId)
end

-- Add an index of when the pause was added.
redis.call("ZREM", keyPauseAddIdx, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZREM", keyPauseExpIdx, pauseID)

return 0
