--[[

Output:
  0: Successfully finalized
  1: Function ended

]]

local metadataKey = KEYS[1]

if redis.call("HINCRBY", metadataKey, "pending", -1) ~= 0 then
	return 0;
end

-- Set status to complete
redis.call("HSET", metadataKey, "status", 1)

return 1;
