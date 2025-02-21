--[[

Updates pending steps for a run.

Output:
  0: Success

]]

local keyStepsPending = KEYS[1]
local stepsPending = cjson.decode(ARGV[1])

redis.call("DEL", keyStepsPending)
if #stepsPending > 0 then
    redis.call("SADD", keyStepsPending, unpack(stepsPending))
end

return 0

