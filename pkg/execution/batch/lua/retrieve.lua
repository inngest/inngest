--
-- Retrieves the full batch with batchID
--

local batchKey = KEYS[1]

return redis.call("LRANGE", batchKey, 0, -1)
