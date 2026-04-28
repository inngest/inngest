---
--- Deletes the provided keys
---

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("DEL", key)
  end
end

return 0
