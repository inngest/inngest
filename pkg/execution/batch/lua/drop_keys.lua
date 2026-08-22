---
--- Deletes the provided keys
---

local batchExists = redis.call("EXISTS", KEYS[1])

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("DEL", key)
  end
end

return batchExists
