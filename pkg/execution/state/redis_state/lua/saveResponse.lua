--[[

Saves a response for a step.  This automatically creates history entries
depending on the response being saved.

Input:
  - 1 if the response is an error
  - 1 if the response is final
Output:
  0: Successfully saved response

]]

local actionKey   = KEYS[1]
local errorKey    = KEYS[2]
local metadataKey = KEYS[3]
local historyKey  = KEYS[4]
local stackKey 	  = KEYS[5]

local data    = ARGV[1]
local stepID  = ARGV[2]
local isError = tonumber(ARGV[3])
local isFinal = tonumber(ARGV[4])
local isStepError = tonumber(ARGV[5])
local stepLog = ARGV[6] -- The step log.
local failLog = ARGV[7] -- An optional fail log, if the error is final
local logTime = tonumber(ARGV[8]) -- The timestamp for the log, unix milliseconds

if isError == 0 or (isStepError == 1 and isFinal == 1) then
	if redis.call("HEXISTS", actionKey, stepID) == 1 then
		return -1
	end

	-- Save the step output under step data.
	redis.call("HSET", actionKey, stepID, data)
	redis.call("ZADD", historyKey, logTime, stepLog)
	return tonumber(redis.call("RPUSH", stackKey, stepID))
end

-- Set the step error key.
redis.call("HSET", errorKey, stepID, data)
redis.call("ZADD", historyKey, logTime, stepLog)
if isFinal == 0 then
	return tonumber(redis.call("LLEN", stackKey))
end

redis.call("HINCRBY", metadataKey, "pending", -1)
redis.call("HSET", metadataKey, "status", 2)  -- Mark as failed
redis.call("ZADD", historyKey, logTime+1, failLog) -- The function failed log
return tonumber(redis.call("RPUSH", stackKey, stepID)) -- Mutate the stack for permanent final errors
