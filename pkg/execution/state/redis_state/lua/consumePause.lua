--[[

Consumes a pause.

Output:
  0: Successfully consumed

]]

local actionKey     = KEYS[1]
local stackKey      = KEYS[2]
local keyMetadata   = KEYS[3]

local pauseDataKey = ARGV[1] -- used to set data in run state store
local pauseDataVal = ARGV[2] -- data to set

if actionKey ~= nil and pauseDataKey ~= "" then
  -- idempotency check: only ever consume a pause once
	local existingPos = redis.call("LPOS", stackKey, pauseDataKey)
	if existingPos ~= nil then
    return 0
  end

	redis.call("RPUSH", stackKey, pauseDataKey)
	redis.call("HSET", actionKey, pauseDataKey, pauseDataVal)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HINCRBY", keyMetadata, "state_size", #pauseDataVal)
end

return 0
