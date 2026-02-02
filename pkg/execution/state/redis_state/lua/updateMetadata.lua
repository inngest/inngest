-- UpdateMetadata updates a run's metadata.

local keyMetadata = KEYS[1]

local die      = ARGV[1] -- disable immediate execution
local rv       = ARGV[2] -- request version
local sat      = ARGV[3] -- started at
local hasAI    = ARGV[4] -- has AI

local function is_field_empty(field, emptyval)
  local val = redis.call("HGET", keyMetadata, field)
   -- 0.0 is for lua 5.4 and redis nil is lua false
  return val == nil or val == emptyval or val == false or val == "" or val == "0.0" or val == "0" or val == 0.0 or val == 0
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
