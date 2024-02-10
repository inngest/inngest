---
--- Expires the provided keys
---

local timeout = tonumber(ARGV[1]) -- timeout in seconds

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("EXPIRE", key, timeout)
  end
end

return 0
