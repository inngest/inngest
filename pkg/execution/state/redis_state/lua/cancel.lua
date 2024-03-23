--[[

Output:
  0: Successfully cancelled
  1: Function already completed
  2: Function already errored
  3: Function already cancelled

]]

local metadataKey = KEYS[1]

local value = tonumber(redis.call("HGET", metadataKey, "status"))

-- If run has ended (cancelled, completed, failed). We only allow cancellation
-- of scheduled or running functions
if value ~= 0 and value ~= 5 then
	-- Return the function status as an error
	return value;
end

redis.call("HSET", metadataKey, "status", 3)

return 0;
