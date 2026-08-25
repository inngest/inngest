---
--- Deletes the provided keys
---

-- DeleteKeys passes the batch list first and its metadata hash second.
local batchKey = KEYS[1]
local batchExists = redis.call("EXISTS", batchKey)

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("DEL", key)
  end
end

return batchExists
