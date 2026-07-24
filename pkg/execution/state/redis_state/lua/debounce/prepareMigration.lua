--[[

Creates a new debounce for the given function, or returns -1 if a
debounce currently exists.

Return values:
- [0] - No existing debounce
- [1, debounceID (string)] if debounceItem.t is not set
- [1, debounceID (string), debounce timeout (unix millis)]
]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key
local keyDebounceMigrating = KEYS[3]

local newDebounceID = ARGV[1]

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
local pointerTTL = redis.call("PTTL", keyPtr)

-- Prevent the next prepareMigration() call from finding the same debounce again. It will immediately
-- create/update a debounce on the primary.
-- Note: This does not prevent the debounce from running on the secondary cluster on timeout.
if pointerTTL ~= nil and tonumber(pointerTTL) > 0 then
	redis.call("SET", keyPtr, newDebounceID, "PX", pointerTTL)
else
	redis.call("SET", keyPtr, newDebounceID)
end

-- Prevent the timeout job from running, in case we are racing with StartExecution().
-- We drop the debounce state and timeout item immediately after prepareMigration(), this is just a protection against data races.
redis.call("HSET", keyDebounceMigrating, existingDebounceID, 1)

-- If timeout is not provided, only return debounce ID
if debounceItem.t == nil or debounceItem.t <= 0 then
	return { 1, existingDebounceID, 0, pointerTTL }
end

-- Return debounce ID and current timeout (carried over from first event)
return { 1, existingDebounceID, debounceItem.t, pointerTTL }
