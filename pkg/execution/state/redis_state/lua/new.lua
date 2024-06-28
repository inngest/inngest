--[[

Output:
  0: Stored successfully
  1: Run ID already exists

]]

local eventsKey = KEYS[1]
local metadataKey = KEYS[2]
local stepKey = KEYS[3]

local events = ARGV[1]
local metadata = ARGV[2]
local steps = ARGV[3]

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
redis.call("HINCRBY", metadataKey, "event_size", #events)

return 0
