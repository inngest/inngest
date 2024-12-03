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
local keyStepInputs = KEYS[4]

local stepID = ARGV[1]
local outputData = ARGV[2]

if redis.call("HEXISTS", keyStep, stepID) == 1 then
  return -1
end

-- If we're saving a response for a step that previously had input, remove the
-- input from the state size in order to keep it as accurate as possible.
local inputData = redis.call("HGET", keyStepInputs, stepID)
local stateSizeDelta = #outputData
if inputData then
  stateSizeDelta = stateSizeDelta - #inputData
end
redis.call("HINCRBY", keyMetadata, "state_size", stateSizeDelta)
redis.call("HINCRBY", keyMetadata, "step_count", 1)

redis.call("HSET", keyStep, stepID, outputData)
redis.call("RPUSH", keyStack, stepID)

return 0
