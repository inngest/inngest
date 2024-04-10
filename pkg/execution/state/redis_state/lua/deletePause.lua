--[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseKey      = KEYS[1]
local pauseStepKey  = KEYS[2]
local pauseEventKey = KEYS[3]
local pauseInvokeKey = KEYS[4]

local pauseID       = ARGV[1]
local invokeCorrelationId = ARGV[2]

redis.call("HDEL", pauseEventKey, pauseID)
redis.call("DEL", pauseKey)
redis.call("DEL", pauseStepKey)

if invokeCorrelationId ~= false and invokeCorrelationId ~= "" and invokeCorrelationId ~= nil then
  redis.call("HDEL", pauseInvokeKey, invokeCorrelationId)
end

return 0
