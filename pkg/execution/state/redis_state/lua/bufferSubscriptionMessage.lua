--[[

Buffers a message of a step.subscribe subscription that arrived while its
reader was not waiting. The message becomes a step only when the reader asks
for it (see flush). Redelivered events are buffered once, and an event that is
already a step is not buffered at all.

KEYS[1] - the run's step results (a saved message is a field there)
KEYS[2] - the run's buffered messages (list)
KEYS[3] - the run's buffered message index (hash by step ID)
ARGV[1] - the step ID of the message
ARGV[2] - the entry to append: the subscription hash, step ID and value as JSON
ARGV[3] - the most messages that may wait in the buffer

Output:
  1: buffered
  0: already a step or already buffered
  -1: the buffer is full, nothing was buffered

]]

if redis.call("HEXISTS", KEYS[1], ARGV[1]) == 1 then
  return 0
end

if redis.call("HEXISTS", KEYS[3], ARGV[1]) == 1 then
  return 0
end

if redis.call("LLEN", KEYS[2]) >= tonumber(ARGV[3]) then
  return -1
end

redis.call("HSET", KEYS[3], ARGV[1], "1")
redis.call("RPUSH", KEYS[2], ARGV[2])

return 1
