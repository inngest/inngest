-- UpdateMetadata updates a run's metadata.

local keyMetadata = KEYS[1]

local ctx      = ARGV[1]
local debugger = ARGV[2]
local die      = ARGV[3] -- disable immediate execution
local rv       = ARGV[4] -- request version
local spanid   = ARGV[5] -- spanID
local sat      = ARGV[6] -- started at

local function is_field_empty(field, emptyval)
  local val = redis.call("HGET", keyMetadata, field)
  return val == nil or val == emptyval
end

redis.call("HSET", keyMetadata, "ctx", ctx)
redis.call("HSET", keyMetadata, "die", die)
redis.call("HSET", keyMetadata, "debugger", debugger)
redis.call("HSET", keyMetadata, "rv", rv)

-- only update the spanID if the existing value is empty
if is_field_empty("sid", "") then
  redis.call("HSET", keyMetadata, "sid", spanid)
end

if is_field_empty("sat", "0") then
  redis.call("HSET", keyMetadata, "sat", sat)
end

return 0
