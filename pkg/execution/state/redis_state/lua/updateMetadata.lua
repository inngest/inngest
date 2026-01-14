-- UpdateMetadata updates a run's metadata.

local keyMetadata = KEYS[1]

local die      = ARGV[1] -- disable immediate execution
local rv       = ARGV[2] -- request version
local sat      = ARGV[3] -- started at
local hasAI    = ARGV[4] -- has AI

local function is_field_empty(field, emptyval)
  local val = redis.call("HGET", keyMetadata, field)
  return val == nil or val == emptyval
end

redis.call("HSET", keyMetadata, "die", die)
redis.call("HSET", keyMetadata, "rv", rv)

if is_field_empty("sat", "0") then
  redis.call("HSET", keyMetadata, "sat", sat)
end

if hasAI == "1" then
  redis.call("HSET", keyMetadata, "hasAI", hasAI)
end

return 0
