--[[

Marks a step.subscribe subscription of a run as waiting: a function thread has
nothing to read and nothing is running for it. Does nothing if the run is gone.

KEYS[1] - the run's waiting hash
KEYS[2] - the run's metadata
ARGV[1] - the subscription hash
ARGV[2] - the token of this wait

Output:
  0: The run does not exist
  1: Marked

]]

if redis.call("EXISTS", KEYS[2]) == 0 then
  return 0
end

redis.call("HSET", KEYS[1], ARGV[1], ARGV[2])

return 1
