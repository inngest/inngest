--[[
Peek returns account IDs from the global account ZSET in order via their index.
]]

local keyGlobalAccountPointer = KEYS[1]

local peekUntilMS  = tonumber(ARGV[1])
local limit        = tonumber(ARGV[2])
local sequential   = tonumber(ARGV[3])

local peekUntil    = math.ceil(peekUntilMS / 1000)

local count = redis.call("ZCOUNT", keyGlobalAccountPointer, "-inf", peekUntil)
local offset = 0

if count > limit and sequential == 0 then
	math.randomseed(peekUntilMS);
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count-limit)+1) - 1
end

return redis.call("ZRANGE", keyGlobalAccountPointer, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
