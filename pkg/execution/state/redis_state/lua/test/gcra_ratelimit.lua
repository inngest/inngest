local throttleKey = ARGV[1]
local currentTime = tonumber(ARGV[2])
local period_ms = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local burst = tonumber(ARGV[5])

-- $include(gcra.lua)

local throttleResult = gcra(throttleKey, currentTime, period_ms, limit, burst)

-- not allowed
if throttleResult[1] == false then
	return 0
end

-- burst used
if throttleResult[2] then
	return 2
end

-- allowed
return 1
