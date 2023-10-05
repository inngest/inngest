--[[

Updates a debounce to use new data.

Return values:
- 0 (int): OK

]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key

local debounceID = ARGV[1] 
local debounce   = ARGV[2]
local ttl        = tonumber(ARGV[3])

-- Set the fn -> debounce ID pointer
redis.call("SETEX", keyPtr, ttl, debounceID)
redis.call("HSET", keyDbc, debounceID, debounce)

return 0
