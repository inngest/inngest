local gcraKey = ARGV[1]
local nowMS = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local burst = tonumber(ARGV[4])
local period = tonumber(ARGV[5])
local capacity = tonumber(ARGV[6])

-- $include(gcra.lua)

if capacity > 0 then
	return cjson.encode(gcraUpdate(gcraKey, nowMS, period, limit, burst, capacity))
end

return cjson.encode(gcraCapacity(gcraKey, nowMS, period, limit, burst))
