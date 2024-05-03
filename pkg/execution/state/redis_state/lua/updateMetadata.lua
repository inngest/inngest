-- UpdateMetadata updates a run's metadata.

local keyMetadata = KEYS[1]

local die      = ARGV[1] -- disable immediate execution
local rv       = ARGV[2] -- request version
local sat      = ARGV[3] -- started at

local function is_field_empty(field, emptyval)
  local val = redis.call("HGET", keyMetadata, field)
  return val == nil or val == emptyval
end

redis.call("HSET", keyMetadata, "die", die)
redis.call("HSET", keyMetadata, "rv", rv)

if is_field_empty("sat", "0") then
  redis.call("HSET", keyMetadata, "sat", sat)
end

return 0
