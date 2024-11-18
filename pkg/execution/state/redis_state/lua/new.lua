--[[

Output:
  0: Stored successfully
  1: Run ID already exists

]]

local eventsKey = KEYS[1]
local metadataKey = KEYS[2]
local stepKey = KEYS[3]
local stepStackKey = KEYS[4]
local stepInputsKey = KEYS[5]

local events = ARGV[1]
local metadata = ARGV[2]
local steps = ARGV[3]
local stepInputs = ARGV[4]

-- Save all metadata
local metadataJson = cjson.decode(metadata)
for k, v in pairs(metadataJson) do
  if k == "ctx" or k == "id" then
    v = cjson.encode(v)
  end
  redis.call("HSET", metadataKey, k, tostring(v))
end

-- Save pre-memoized steps
if steps ~= nil and #steps > 0 then
  local stepsArray = cjson.decode(steps)

  for _, step in ipairs(stepsArray) do
    redis.call("HSET", stepKey, step.id, cjson.encode(step.data))
    redis.call("RPUSH", stepStackKey, step.id)
  end
end

-- Save pre-memoized step inputs
if stepInputs ~= nil and #stepInputs > 0 then
  local stepInputsArray = cjson.decode(stepInputs)

  for _, stepInput in ipairs(stepInputsArray) do
    redis.call("HSET", stepInputsKey, stepInput.id, cjson.encode(stepInput.data))
  end
end

-- Save events
redis.call("SETNX", eventsKey, events)
redis.call("HINCRBY", metadataKey, "event_size", #events)

return 0
