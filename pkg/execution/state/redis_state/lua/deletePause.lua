--[[

Deletes a pause.

Output:
  0: Successfully deleted

]]

local pauseKey      = KEYS[1]
local pauseStepKey  = KEYS[2]
local pauseEventKey = KEYS[3]
local pauseID       = ARGV[1]

redis.call("HDEL", pauseEventKey, pauseID)
redis.call("DEL", pauseKey)
redis.call("DEL", pauseStepKey)
return 0
