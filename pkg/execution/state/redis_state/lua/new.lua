--[[

Output:
  0: Stored successfully
  1: Run ID already exists

]]

local eventsKey = KEYS[1]
local metadataKey = KEYS[2]
local stepKey = KEYS[3]
local stepStackKey = KEYS[4]

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

local stepCount = 0
local stateSize = 0

if steps ~= nil and #steps > 0 then
  local stepsArray = cjson.decode(steps)
  stepCount = #stepsArray

  for _, step in ipairs(stepsArray) do
    local stepData = cjson.encode(step.data)
    stateSize = stateSize + #stepData

    redis.call("HSET", stepKey, step.id, stepData)
    redis.call("RPUSH", stepStackKey, step.id)
  end
end

redis.call("SETNX", eventsKey, events)
redis.call("HINCRBY", metadataKey, "event_size", #events)

if stepCount > 0 then
  redis.call("HINCRBY", metadataKey, "step_count", stepCount)
  redis.call("HINCRBY", metadataKey, "state_size", stateSize)
end

return 0
