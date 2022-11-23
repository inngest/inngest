--[[

Output:
  0: Successfully finalized
  1: Function ended

]]

local metadataKey = KEYS[1]
local historyKey  = KEYS[2]

local funcLog = ARGV[1]
local logTime = tonumber(ARGV[2])

if redis.call("HINCRBY", metadataKey, "pending", -1) ~= 0 then
	return 0;
end

-- Only transition to complete if the function hasn't been cancelled or marked as failed.
if tonumber(redis.call("HGET", metadataKey, "status")) == 0 then
	redis.call("HSET", metadataKey, "status", 1)
	redis.call("ZADD", historyKey, logTime, funcLog)
	return 1;
end

return 0;
