--[[

Tests and clears the waiting flag of a step.subscribe subscription. Only the
caller that finds the flag set gets its token, so only one caller wakes the run.

KEYS[1] - the run's waiting hash
ARGV[1] - the subscription hash

Output:
  nil: The flag was not set
  the token of the wait that set it

]]

local token = redis.call("HGET", KEYS[1], ARGV[1])
if not token then
  return false
end

redis.call("HDEL", KEYS[1], ARGV[1])

return token
