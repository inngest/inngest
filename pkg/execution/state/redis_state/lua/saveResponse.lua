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
local stackKey 	  = KEYS[4]

local data    = ARGV[1]
local stepID  = ARGV[2]
local isError = tonumber(ARGV[3])
local isFinal = tonumber(ARGV[4])

if isError == 0 then
	if redis.call("HEXISTS", actionKey, stepID) == 1 then
		return -1
	end

	-- Save the step output under step data.
	redis.call("HSET", actionKey, stepID, data)
	return tonumber(redis.call("RPUSH", stackKey, stepID))
end

-- Set the step error key.
redis.call("HSET", errorKey, stepID, data)
if isFinal == 0 then
	return tonumber(redis.call("LLEN", stackKey))
end

redis.call("HINCRBY", metadataKey, "pending", -1)
redis.call("HSET", metadataKey, "status", 2)  -- Mark as failed
return tonumber(redis.call("RPUSH", stackKey, stepID)) -- Mutate the stack for permanent final errors
