--[[

Leases a pause atomically, if the pause is not already leased or the lease has expired.

Output:
  0: Leased
  1: Already leased
  2: Pause not found

]]

-- The pause ID is always provided as a key
local pauseID = KEYS[1]
-- The current time and lease time are provided as unix timestamps in MS
local currentTime = tonumber(ARGV[1])
local leaseTime = tonumber(ARGV[2])

if redis.call("EXISTS", pauseID) ~= 1 then
	return 2
end

local pause = cjson.decode(redis.call("GET", pauseID))

if pause.leasedUntil ~= nil and pause.leasedUntil > currentTime then
	-- unable to lease
	return 1
end

pause.leasedUntil = leaseTime

local ttl = redis.call("TTL", pauseID)
redis.call("SETEX", pauseID, ttl, cjson.encode(pause))

return 0
