--[[

Updates pending steps for a run.

Uses SADD instead of destructive DEL + SADD so that a re-delivered
discovery edge cannot re-add steps that have already completed.
Each step is checked against the actions hash; only steps that have
NOT been completed (i.e. do not exist in the actions hash) are added.

Output:
  0: Success

]]

local keyStepsPending = KEYS[1]
local actionKey       = KEYS[2]
local stepsPending = cjson.decode(ARGV[1])

for _, step in ipairs(stepsPending) do
  if redis.call("HEXISTS", actionKey, step) == 0 then
    redis.call("SADD", keyStepsPending, step)
  end
end

return 0

