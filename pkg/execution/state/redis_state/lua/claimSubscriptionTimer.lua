--[[

Arbitrates a timed step.subscribe read between its timer and the wake of a
message, with no step saved: whichever claims first runs what follows the read.

A timed read is registered when its wait is reported. A claim on a timer that
was never registered is not a subscription timer and returns an empty string.

KEYS[1] - the run's timer claims hash
ARGV[1] - the timer step ID
ARGV[2] - "register" or "claim"
ARGV[3] - who claims (claim only)

Output:
  register: 1
  claim: the claimant that holds the timer, or "" if it is not registered

]]

if ARGV[2] == "register" then
  redis.call("HSETNX", KEYS[1], ARGV[1], "-")

  return 1
end

local held = redis.call("HGET", KEYS[1], ARGV[1])
if not held then
  return ""
end

if held == "-" then
  redis.call("HSET", KEYS[1], ARGV[1], ARGV[3])

  return ARGV[3]
end

return held
