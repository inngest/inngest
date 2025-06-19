local keyBacklogActiveCheckSet   = KEYS[1]
local keyBacklogActiveCheckCooldown = KEYS[2]

local backlogID       = ARGV[1]
local nowMS           = tonumber(ARGV[2])
local cooldownSeconds = tonumber(ARGV[3])

redis.call("ZREM", keyBacklogActiveCheckSet, backlogID)
redis.call("SET", keyBacklogActiveCheckCooldown, nowMS, "EX", cooldownSeconds)

return 0
