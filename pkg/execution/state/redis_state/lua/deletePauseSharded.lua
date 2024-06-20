  --[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseKey      = KEYS[1]
local pauseStepKey  = KEYS[2]
local keyRunPauses  = KEYS[3]

local pauseID       = ARGV[1]
local invokeCorrelationId = ARGV[2]

redis.call("DEL", pauseKey)
redis.call("DEL", pauseStepKey)

-- SREM to remove the pause for this run
redis.call("SREM", keyRunPauses, pauseID)

return 0
