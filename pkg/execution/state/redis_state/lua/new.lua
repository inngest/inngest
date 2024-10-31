--[[

Output:
  0: Stored successfully
  1: Run ID already exists

]]

local eventsKey = KEYS[1]
local metadataKey = KEYS[2]
local stepKey = KEYS[3]
-- local stepInputKey = KEYS[4]
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

-- TODO Must also set stack and increment `step_count` and `state_size` in
-- metadata. How do we know what the stack was at this point?
-- if steps ~= nil and steps ~= "" then
--   local stepsJson = cjson.decode(steps)

--   for k, v in pairs(stepsJson) do
--     redis.call("HSET", stepKey, k, cjson.encode(v))
--   end
-- end
if steps ~= nil and #steps > 0 then
  local stepsArray = cjson.decode(steps)

  for _, step in ipairs(stepsArray) do
    redis.call("HSET", stepKey, step.id, cjson.encode(step.data))
    redis.call("RPUSH", stepStackKey, step.id)
  end
end

-- if stepInputs ~= nil and stepInputs ~= "" then
--   local stepInputsJson = cjson.decode(stepInputs)

--   for k, v in pairs(stepInputsJson) do
--     redis.call("HSET", stepInputKey, k, cjson.encode(v))
--   end
-- end

redis.call("SETNX", eventsKey, events)
redis.call("HINCRBY", metadataKey, "event_size", #events)

return 0
