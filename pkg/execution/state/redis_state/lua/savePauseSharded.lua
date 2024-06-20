-- [[
--
-- Output:
--   0: Successfully saved pause
--   1: Pause already exists
-- ]]

local pauseKey    = KEYS[1]
local keyRunPauses   = KEYS[2]

local pause          = ARGV[1]
local pauseID        = ARGV[2]
local extendedExpiry = tonumber(ARGV[3])

if redis.call("SETNX", pauseKey, pause) == 0 then
	return 1
end

redis.call("EXPIRE", pauseKey, extendedExpiry)

-- SADD to store the pause for this run
redis.call("SADD", keyRunPauses, pauseID)

return 0
