--[[

Creates a new debounce for the given function, or returns -1 if a
debounce currently exists.

Return values:
- "0" (string): Success
- "$ID": The existing debounce ID

]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key

local debounceID = ARGV[1] 
local debounce   = ARGV[2]
local ttl        = tonumber(ARGV[3])

local existing = redis.call("GET", keyPtr)
if existing ~= nil and existing ~= false then
	-- A debounce for this function exists.  Check that this is in the map, first.
	local found = redis.call("HEXISTS", keyDbc, debounceID)
	if found == 1 then
		-- The debounce exists in the map, too.  Return this debounce.
		return existing
	end
end

-- Set the fn -> debounce ID pointer
redis.call("SETEX", keyPtr, ttl, debounceID)
-- Set debounce info
redis.call("HSET", keyDbc, debounceID, debounce)

-- TODO: Ideally, enqueue would be atomic here.  We should make enqueue a function.

return "0"
