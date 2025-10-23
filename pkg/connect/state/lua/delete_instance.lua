--[[

Decrement the instanceId counter when a lease is deleted.
- Decrements the counter atomically
- Removes the counter key when it reaches 0

Output:
  0: Counter decremented successfully (errors are ignored)

]]

local keyInstanceCounter = KEYS[1]

local instanceID = ARGV[1]

-- Decrement the instanceId counter
if instanceID ~= nil and instanceID ~= "" then
	local currentCount = tonumber(redis.call("GET", keyInstanceCounter))

	-- Only decrement if counter exists and is greater than 0
	if currentCount ~= nil and currentCount > 0 then
		redis.call("DECR", keyInstanceCounter)

		-- If count is now 0, delete the counter key
		local newCount = tonumber(redis.call("GET", keyInstanceCounter))
		if newCount == 0 then
			redis.call("DEL", keyInstanceCounter)
		end
	end
end

return 0
