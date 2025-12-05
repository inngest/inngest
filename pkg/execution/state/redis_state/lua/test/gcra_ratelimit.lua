local throttleKey = ARGV[1]
local currentTime = tonumber(ARGV[2])
local period_ms = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local burst = tonumber(ARGV[5])

-- $include(gcra.lua)

local throttleResult = gcra(throttleKey, currentTime, period_ms, limit, burst)

-- Convert boolean to integer for Redis
if throttleResult then
  return 1
else
  return 0
end
