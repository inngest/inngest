local keyActiveCheckSet   = KEYS[1]
local keyActiveCheckCooldown = KEYS[2]

local pointer         = ARGV[1]
local nowMS           = tonumber(ARGV[2])
local cooldownSeconds = tonumber(ARGV[3])

redis.call("ZREM", keyActiveCheckSet, pointer)
redis.call("SET", keyActiveCheckCooldown, nowMS, "EX", cooldownSeconds)

return 0
