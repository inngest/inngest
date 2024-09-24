---
--- Deletes the provided keys
---

local timeout = tonumber(ARGV[1]) -- timeout in seconds

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("DEL", key)
  end
end

return 0
