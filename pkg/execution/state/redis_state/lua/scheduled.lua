--[[

Increases the waitgroup count for a step when scheduling a new step run

]]

local metadataKey = KEYS[1]
local historyKey  = KEYS[2]

local stepLog = ARGV[1]
local logTime = tonumber(ARGV[2])
local disableImmExec = tonumber(ARGV[3])

redis.call("HINCRBY", metadataKey, "pending", 1)
redis.call("ZADD", historyKey, logTime, stepLog)

if disableImmExec == 1 then
  redis.call("HSET", metadataKey, "disableImmediateExecution", 1)
end

return 0
