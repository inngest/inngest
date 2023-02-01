--[[

Output:
  0: Successfully cancelled
  1: Function already completed
  2: Function already errored
  3: Function already cancelled

]]

local metadataKey = KEYS[1]
local historyKey  = KEYS[2]

local historyLog  = ARGV[1]
local logTime     = tonumber(ARGV[2])

local value = tonumber(redis.call("HGET", metadataKey, "status"))
if value ~= 0 then
	-- Return the function status as an error
	return value;
end

redis.call("HSET", metadataKey, "status", 3)
redis.call("ZADD", historyKey, logTime, historyLog)

return 0;
