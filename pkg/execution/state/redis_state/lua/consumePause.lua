--[[

Consumes a pause.

Output:
  -1: Pause already consumed
  0: Successfully consumed, no pending steps
  1: Successfully consumed, at least one pending step remains

]]

local actionKey     = KEYS[1]
local stackKey      = KEYS[2]
local keyMetadata   = KEYS[3]
local keyStepsPending = KEYS[4]

local pauseDataKey = ARGV[1] -- used to set data in run state store
local pauseDataVal = ARGV[2] -- data to set

if actionKey ~= nil and pauseDataKey ~= "" then
  -- idempotency check: only ever consume a pause once
  if redis.call("HEXISTS", actionKey, pauseDataKey) == 1 then
    return -1
  end

  redis.call("RPUSH", stackKey, pauseDataKey)
  redis.call("HSET", actionKey, pauseDataKey, pauseDataVal)
  redis.call("HINCRBY", keyMetadata, "step_count", 1)
  redis.call("HINCRBY", keyMetadata, "state_size", #pauseDataVal)
end

redis.call("SREM", keyStepsPending, pauseDataKey)
return redis.call("SCARD", keyStepsPending) > 0 and 1 or 0
