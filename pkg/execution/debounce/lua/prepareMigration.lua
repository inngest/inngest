--[[

Creates a new debounce for the given function, or returns -1 if a
debounce currently exists.

Return values:
- [0] - No existing debounce
- [1, debounceID (string), debounce timeout (unix millis)]
]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key

local newDebounceID = ARGV[1]
local currentTime 	= tonumber(ARGV[2]) -- in ms

local existingDebounceID = redis.call("GET", keyPtr)
if existingDebounceID == nil or existingDebounceID == false then
	-- No existing ID
	return { 0 }
end

local existingDebounceItemStr = redis.call("HGET", keyDbc, existingDebounceID)
if existingDebounceItemStr == false then
	-- No existing debounce
	return { 0 }
end

local debounceItem = cjson.decode(existingDebounceItemStr)

-- Prevent this debounce from running on the default cluster (we're moving it to the new system queue)
redis.call("SET", debouncePointerKey, newDebounceID)

-- Return debounce ID and current timeout (carried over from first event)
return { 1, debounceID, debounceItem.t }
