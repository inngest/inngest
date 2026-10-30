--[[

Clears the waiting flag of a step.subscribe subscription if it still holds the
given token. A flag that holds another token, or none, is left alone.

KEYS[1] - the run's waiting hash
ARGV[1] - the subscription hash
ARGV[2] - the token

Output:
  0: Nothing was cleared
  1: Cleared

]]

local token = redis.call("HGET", KEYS[1], ARGV[1])
if token == ARGV[2] then
  redis.call("HDEL", KEYS[1], ARGV[1])
  return 1
end

return 0
