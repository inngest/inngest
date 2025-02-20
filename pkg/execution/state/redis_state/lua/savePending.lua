-- SetPending updates pending steps for a run

local keyStepsPending = KEYS[1]
local stepsPending = cjson.decode(ARGV[1])

redis.call("DEL", keyStepsPending)
if #stepsPending > 0 then
    redis.call("SADD", keyStepsPending, unpack(stepsPending))
end

return 0

