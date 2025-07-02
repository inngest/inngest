local keyActiveCheckSet   = KEYS[1]
local keyActiveCheckCooldown = KEYS[2]

local pointer         = ARGV[1]
local nowMS           = tonumber(ARGV[2])
local cooldownSeconds = tonumber(ARGV[3])

redis.call("ZREM", keyActiveCheckSet, pointer)

if cooldownSeconds > 0 then
  redis.call("SET", keyActiveCheckCooldown, nowMS, "EX", cooldownSeconds)
end

return 0
