--[[

Output:
  0: Stored successfully
  1: Run ID already exists

]]

local idempotencyKey = KEYS[1]
local eventsKey = KEYS[2]
local metadataKey = KEYS[3]
local stepKey = KEYS[4]
local logKey = KEYS[5]

local events = ARGV[1]
local metadata = ARGV[2]
local steps = ARGV[3]
local log = ARGV[4]
local logScore = tonumber(ARGV[5])

if redis.call("SETNX", idempotencyKey, "") == 0 then
  -- If this key exists, everything must've been initialised, so we can exit early
  return 1
end

local metadataJson = cjson.decode(metadata)
for k, v in pairs(metadataJson) do
  if k == "ctx" or k == "id" then
    v = cjson.encode(v)
  end
  redis.call("HSET", metadataKey, k, tostring(v))
end

if steps ~= nil and steps ~= "" then
  local stepsJson = cjson.decode(steps)

  for k, v in pairs(stepsJson) do
    redis.call("HSET", stepKey, k, cjson.encode(v))
  end
end

redis.call("SETNX", eventsKey, events)
redis.call("ZADD", logKey, logScore, log)

return 0
