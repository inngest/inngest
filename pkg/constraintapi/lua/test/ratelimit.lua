local gcraKey = ARGV[1]
local nowNS = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local burst = tonumber(ARGV[4])
local periodNS = tonumber(ARGV[5])
local capacity = tonumber(ARGV[6])

-- $include(helper/gcra.lua)

return cjson.encode(rateLimit(gcraKey, nowNS, periodNS, limit, burst, capacity))
