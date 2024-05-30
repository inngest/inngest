--[[

Consumes a pause.

Output:
  0: Successfully consumed
  1: Pause not found

]]

-- The pause ID is always provided as a key, as is the lease ID.
local pauseKey      = KEYS[1]
local pauseStepKey  = KEYS[2]
local pauseEventKey = KEYS[3]
local pauseInvokeKey = KEYS[4]
local actionKey     = KEYS[5]
local stackKey      = KEYS[6]
local keyMetadata   = KEYS[7]
local keyPauseAddIdx = KEYS[8]
local keyPauseExpIdx = KEYS[9]

local pauseID      = ARGV[1]
local invokeCorrelationId = ARGV[2]
local pauseDataKey = ARGV[3] -- used to set data in run state store
local pauseDataVal = ARGV[4] -- data to set

local pause = redis.call("GET", pauseKey)
if pause == false or pause == nil then
	-- Pause no longer exists.
	if pauseEventKey ~= "" then
		-- Clean up regardless
		redis.call("HDEL", pauseEventKey, pauseID)
		-- Add an index of when the pause was added.
		redis.call("ZREM", keyPauseAddIdx, pauseID)
		-- Add an index of when the pause expires.  This lets us manually
		-- garbage collect expired pauses from the HSET below.
		redis.call("ZREM", keyPauseExpIdx, pauseID)
	end
	return 1
end

redis.call("DEL", pauseKey)
redis.call("DEL", pauseStepKey)

if pauseEventKey ~= "" then
	redis.call("HDEL", pauseEventKey, pauseID)
end

if actionKey ~= nil and pauseDataKey ~= "" then
	redis.call("RPUSH", stackKey, pauseDataKey)
	redis.call("HSET", actionKey, pauseDataKey, pauseDataVal)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HINCRBY", keyMetadata, "state_size", #pauseDataVal)
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
