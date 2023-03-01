-- Checks whether there's capacity in the given concurrency queue, given a limit and the
-- current time in milliseconds.
local function check_concurrency(now_ms, key, limit)
	local count = redis.call("ZCOUNT", key, tostring(now_ms), "+inf")
	return tonumber(limit) - tonumber(count)
end
