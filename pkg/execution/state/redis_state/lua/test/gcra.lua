local gcraKey = ARGV[1]
local nowMS = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local burst = tonumber(ARGV[4])
local period = tonumber(ARGV[5])
local capacity = tonumber(ARGV[6])

-- $include(gcra.lua)

return cjson.encode(applyGCRA(gcraKey, nowMS, period, limit, burst, capacity))
