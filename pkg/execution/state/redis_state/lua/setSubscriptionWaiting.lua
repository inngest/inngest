--[[

Marks a step.subscribe subscription of a run as waiting: a function thread has
nothing to read and nothing is running for it. Does nothing if the run is gone.

A token of a timed read ends in the ID of the read's timer step. When that
timer has been claimed (the timer fired or a message won) the wait is over, and
a late write of its token would overwrite the flag of a newer wait, so nothing
is set.

KEYS[1] - the run's waiting hash
KEYS[2] - the run's metadata
KEYS[3] - the run's timer claims hash
ARGV[1] - the subscription hash
ARGV[2] - the token of this wait

Output:
  0: The run does not exist
  1: Marked
  2: The wait is over, nothing was marked

]]

if redis.call("EXISTS", KEYS[2]) == 0 then
  return 0
end

local timer = string.match(ARGV[2], "^[^.]+%.([^.]+)%.%d+$")
if timer then
  local held = redis.call("HGET", KEYS[3], timer)
  if held and held ~= "-" then
    return 2
  end
end

redis.call("HSET", KEYS[1], ARGV[1], ARGV[2])

return 1
