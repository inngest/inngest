--[[

Consumes a pause.

Output:
  -1: Pause already consumed
  0: Successfully consumed, no pending steps
  1: Successfully consumed, at least one pending step remains

]]

local actionKey       = KEYS[1]
local stackKey        = KEYS[2]
local keyMetadata     = KEYS[3]
local keyStepsPending = KEYS[4]
local keyIdempotency  = KEYS[5]

local pauseDataKey          = ARGV[1] -- used to set data in run state store
local pauseDataVal          = ARGV[2] -- data to set
local pauseIdempotencyValue = ARGV[3] -- the idempotency key value
local pauseIdempotencyTTL   = tonumber(ARGV[4]) -- TTL of the idempotency key in seconds


if actionKey ~= nil and pauseDataKey ~= "" then
  -- Check if idempotency key exists and get its value
  -- NOTE: We use GET + SET NX separately instead of SET ... NX GET for Garnet compatibility.
  -- Lua scripts are atomic so this is safe.
  local prev = redis.call("GET", keyIdempotency)

  if not prev or prev == false then
    -- Key doesn't exist, set it with NX and expiration
    -- NOTE: We use EX (relative TTL) instead of EXAT (absolute timestamp) for Garnet compatibility
    redis.call("SET", keyIdempotency, pauseIdempotencyValue, "NX", "EX", pauseIdempotencyTTL)

    -- idempotency check: only ever consume a pause once
    if redis.call("HEXISTS", actionKey, pauseDataKey) == 1 then
      return -1
    end

    redis.call("RPUSH", stackKey, pauseDataKey)
    redis.call("HSET", actionKey, pauseDataKey, pauseDataVal)
    redis.call("HINCRBY", keyMetadata, "step_count", 1)
    redis.call("HINCRBY", keyMetadata, "state_size", #pauseDataVal)
    redis.call("SREM", keyStepsPending, pauseDataKey)

  elseif prev ~= pauseIdempotencyValue then
    return -1 -- someone else already consumed it
  end
end

return redis.call("SCARD", keyStepsPending) > 0 and 1 or 0
