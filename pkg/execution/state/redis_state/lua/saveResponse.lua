--[[

Saves a response for a step.  This automatically creates history entries
depending on the response being saved.

Output:
 -1: duplicate response
  0: Successfully saved response

]]

local keyStep     = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack 	  = KEYS[3]

local stepID  = ARGV[1]
local data    = ARGV[2]

if redis.call("HEXISTS", keyStep, stepID) == 1 then
  -- TODO Hmmmmm, this means we're saving input in the state store and it's
  -- travelling to SDKs with every request. This is not good, as input could be
  -- enormous. We should probably filter out input in the marshalled request if
  -- we're not explicitly targeting a step with mutated input and then enable
  -- disableImmediateExecution for every input-specific rerun
  local existingData = cjson.decode(redis.call("HGET", keyStep, stepID))

  if existingData.input == nil then
	  return -1
  end
end

redis.call("HINCRBY", keyMetadata, "step_count", 1)
redis.call("HINCRBY", keyMetadata, "state_size", #data)
redis.call("HSET", keyStep, stepID, data)
redis.call("RPUSH", keyStack, stepID)
return 0
