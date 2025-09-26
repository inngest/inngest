---
--- Deletes the provided keys
---

local batchIdempotenceKey = ARGV[1] -- This key contains all event IDs that were appended for this function
local maxScoreToDrop = ARGV[2] -- This key denotes max score to drop from the idempotence set

redis.call("ZREMRANGEBYSCORE", batchIdempotenceKey, "-inf", maxScoreToDrop)

for i, key in ipairs(KEYS) do
  if i > 0 then
    redis.call("DEL", key)
  end
end

return 0
