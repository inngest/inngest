-- Checks whether there's capacity in the given concurrency queue, given a limit and the
-- current time in milliseconds.
local function check_concurrency(now_ms, key, limit)
	-- TODO: Rnadomly ensure that any expired entries are removed.  Use random outside
	-- of the queue and pass this in, as lua scripts need to be deterministic.
	--
	-- redis.call("ZREMRANGEBYSCORE", key, "-inf", now_ms - 1)

	local count = redis.call("ZCOUNT", key, tostring(now_ms), "+inf")
	return tonumber(limit) - tonumber(count)
end
