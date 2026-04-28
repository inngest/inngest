--[[

  removeItem attempts to remove the queue item from the queue and the loop up map

  0: success
]]

local queueKey     = KEYS[1]
local queueItemKey = KEYS[2]

local itemID = ARGV[1]

redis.call("ZREM", queueKey, itemID)
redis.call("HDEL", queueItemKey, itemID)

-- Clean up any additional index keys (e.g. status indexes) passed by the caller.
for i = 3, #KEYS do
    if KEYS[i] ~= "" then
        redis.call("ZREM", KEYS[i], itemID)
    end
end

return 0
