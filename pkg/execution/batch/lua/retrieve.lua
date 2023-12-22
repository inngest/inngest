--
--Retrieves the full batch from Redis
--

local batchKey = KEYS[1]

return redis.call("LRANGE", batchKey, 0, -1)
