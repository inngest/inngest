--[[

The state of the inboxes (step.subscribe calls) of a run, one field each in one
hash. Every op touches only the field of its own inbox. A field is one of:

  absent:                  idle
  [{i, v}, ...]:           messages buffered, in arrival order (i: step ID, v: the saved value)
  {"waiting": "<token>"}:  a read of this inbox is waiting
  "closed":                the inbox is closed; deliveries are dropped

Buffered and waiting never coexist: a buffered message is flushed, not waited for.

KEYS[1] - the run's inbox hash
KEYS[2] - the run's step results (a saved message is a field there)
KEYS[3] - the run's metadata
KEYS[4] - the run's timer claims hash
ARGV[1] - the op
ARGV[2] - the inbox field (a hash of the subscription ID)
ARGV[3...] - per op

Ops:
  deliver  step ID, value, limit: returns {"closed"}, {"waiting", token}, {"dup"}, {"full"} or {"buffered"}
  wait     token: returns {"norun"}, {"closed"}, {"buffered"}, {"over"} or {"set"}
  get      returns the token of a waiting read, or ""
  list     returns the buffer as JSON, or ""
  ack      step IDs: drops flushed messages from the buffer
  clear    drops the waiting flag of the inbox, leaving a buffer or a close alone
  clearall drops the waiting flag of every inbox of the run
  close    marks the inbox closed

]]

local op = ARGV[1]
local field = ARGV[2]

local function load(f)
  local raw = redis.call("HGET", KEYS[1], f)
  if not raw then
    return nil
  end

  return cjson.decode(raw)
end

local function isClosed(st)
  return st == "closed"
end

local function waitingToken(st)
  if type(st) == "table" and st.waiting then
    return st.waiting
  end

  return nil
end

local function isBuffer(st)
  return type(st) == "table" and st.waiting == nil and #st > 0
end

if op == "deliver" then
  local st = load(field)
  if isClosed(st) then
    return {"closed"}
  end

  local token = waitingToken(st)
  if token then
    return {"waiting", token}
  end

  if redis.call("HEXISTS", KEYS[2], ARGV[3]) == 1 then
    return {"dup"}
  end

  local buf = {}
  if isBuffer(st) then
    buf = st
  end

  for _, entry in ipairs(buf) do
    if entry.i == ARGV[3] then
      return {"dup"}
    end
  end

  if #buf >= tonumber(ARGV[5]) then
    return {"full"}
  end

  table.insert(buf, {i = ARGV[3], v = ARGV[4]})
  redis.call("HSET", KEYS[1], field, cjson.encode(buf))

  return {"buffered"}
end

if op == "wait" then
  if redis.call("EXISTS", KEYS[3]) == 0 then
    return {"norun"}
  end

  local st = load(field)
  if isClosed(st) then
    return {"closed"}
  end

  if isBuffer(st) then
    return {"buffered"}
  end

  -- A token of a timed read carries its timer; a claimed timer ended the wait.
  local timer = string.match(ARGV[3], "^[^.]+%.([^.]+)%.%d+$")
  if timer then
    local held = redis.call("HGET", KEYS[4], timer)
    if held and held ~= "-" then
      return {"over"}
    end
  end

  redis.call("HSET", KEYS[1], field, cjson.encode({waiting = ARGV[3]}))

  return {"set"}
end

if op == "get" then
  return waitingToken(load(field)) or ""
end

if op == "list" then
  local st = load(field)
  if isBuffer(st) then
    return cjson.encode(st)
  end

  return ""
end

if op == "ack" then
  local st = load(field)
  if not isBuffer(st) then
    return 0
  end

  local drop = {}
  for i = 3, #ARGV do
    drop[ARGV[i]] = true
  end

  local kept = {}
  for _, entry in ipairs(st) do
    if not drop[entry.i] then
      table.insert(kept, entry)
    end
  end

  if #kept == 0 then
    redis.call("HDEL", KEYS[1], field)
  else
    redis.call("HSET", KEYS[1], field, cjson.encode(kept))
  end

  return 1
end

if op == "clear" then
  if waitingToken(load(field)) then
    redis.call("HDEL", KEYS[1], field)
  end

  return 1
end

if op == "clearall" then
  for _, f in ipairs(redis.call("HKEYS", KEYS[1])) do
    if waitingToken(load(f)) then
      redis.call("HDEL", KEYS[1], f)
    end
  end

  return 1
end

if op == "close" then
  redis.call("HSET", KEYS[1], field, cjson.encode("closed"))

  return 1
end

return redis.error_reply("unknown subscription state op " .. tostring(op))
