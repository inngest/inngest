local throttleKey = ARGV[1]
local currentTime = tonumber(ARGV[2])
local period_ms = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local burst = tonumber(ARGV[5])
local enableFix = tonumber(ARGV[6]) == 1

-- $include(gcra.lua)

local throttleResult = gcra(throttleKey, currentTime, period_ms, limit, burst, enableFix)

-- not allowed
if throttleResult[1] == 0 then
	return 0
end

-- burst used
if throttleResult[2] then
	return 2
end

-- allowed
return 1
