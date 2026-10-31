--[[

The state of the inboxes (step.subscribe calls) of a run: one hash, one field per
inbox. Every op touches only the field of its own inbox, and a field is exactly
one of:

  absent:                    idle
  [{i, v}, ...]:             messages buffered, in arrival order (i: step ID, v: value)
  {"waiting": "<op id>"}:    a next op of this inbox is waiting
  "closed":                  the inbox is closed; deliveries are dropped

A buffer and a waiting op never coexist: buffered messages are flushed, not waited for.

KEYS[1] - the run's inbox hash
KEYS[2] - the run's step results
KEYS[3] - the run's metadata
KEYS[4] - the run's stack
ARGV[1] - the op
ARGV[2] - the inbox field
ARGV[3...] - per op

Limits, where a message becomes a step: the last four ARGV of deliver, next, flush
and close are the most messages one inbox keeps, the most messages one run keeps,
the run's step limit and the run's state size limit. A message that would break one
is dropped with everything after it, and the inbox closes: the closed step is saved
under its own ID and the field is "closed".

Ops (the result is an array):
  deliver  step ID, value, limits: {"norun"}, {"closed"}, {"buffered"}, {"dup"},
           {"wake", op id} (saved as a step, the op waits) or {"limit", reason, op id}
  next     op id, limits: {"norun"}, {"closed"}, {"waiting"}, {"flushed", n} or
           {"limit", reason}
  flush    limits: {"norun"}, {"flushed", n, total} or {"limit", reason, total}; total is the
           number of messages of the inbox that are steps
  close    limits: as flush, then the inbox is closed
  timeout  op id: {1} when the timer won (null is saved under the op id), else {0}
  clear    op id: drops the waiting flag when it holds this op id

]]

local op = ARGV[1]
local field = ARGV[2]

local messagePrefix = "inngest-sub:"
local closedStep = messagePrefix .. "closed:" .. field

local function load()
  local raw = redis.call("HGET", KEYS[1], field)
  if not raw then
    return nil
  end

  return cjson.decode(raw)
end

local function waitingOp(st)
  if type(st) == "table" and st.waiting then
    return st.waiting
  end

  return nil
end

local function buffer(st)
  if type(st) == "table" and st.waiting == nil and #st > 0 then
    return st
  end

  return {}
end

local function save(id, data)
  redis.call("HINCRBY", KEYS[3], "state_size", #data)
  redis.call("HINCRBY", KEYS[3], "step_count", 1)
  redis.call("HSET", KEYS[2], id, data)
  redis.call("RPUSH", KEYS[4], id)
end

-- messageLimit reports which limit one more message of this inbox would break.
local function messageLimit(size, a)
  local perInbox, perRun = tonumber(a[1]), tonumber(a[2])
  local stepLimit, stateLimit = tonumber(a[3]), tonumber(a[4])
  local inboxPrefix = messagePrefix .. field .. ":"

  local inRun, inInbox = 0, 0
  for _, id in ipairs(redis.call("LRANGE", KEYS[4], 0, -1)) do
    if string.sub(id, 1, #messagePrefix) == messagePrefix and string.sub(id, 1, #messagePrefix + 7) ~= messagePrefix .. "closed:" then
      inRun = inRun + 1
      if string.sub(id, 1, #inboxPrefix) == inboxPrefix then
        inInbox = inInbox + 1
      end
    end
  end

  local stepCount = tonumber(redis.call("HGET", KEYS[3], "step_count")) or 0
  local stateSize = tonumber(redis.call("HGET", KEYS[3], "state_size")) or 0

  if inInbox >= perInbox then
    return "the subscription saved " .. perInbox .. " messages"
  elseif inRun >= perRun then
    return "the run's subscriptions saved " .. perRun .. " messages"
  elseif inRun >= math.floor(stepLimit / 2) then
    return "messages took half of the run's step limit of " .. stepLimit
  elseif stepCount >= stepLimit then
    return "the run reached its step limit of " .. stepLimit
  elseif stateSize + size > math.floor(stateLimit / 4) * 3 then
    return "the run's state reached three quarters of its limit of " .. stateLimit .. " bytes"
  end

  return nil
end

-- saveMessage saves a message as a step unless it already is one. It returns the
-- limit it would break instead, when it breaks one.
local function saveMessage(id, value, a)
  if redis.call("HEXISTS", KEYS[2], id) == 1 then
    return nil
  end

  local reason = messageLimit(#value, a)
  if reason then
    return reason
  end

  save(id, value)

  return nil
end

local function closeForLimit(reason)
  if redis.call("HEXISTS", KEYS[2], closedStep) == 0 then
    save(closedStep, cjson.encode({data = {limit = reason}}))
  end
  redis.call("HSET", KEYS[1], field, cjson.encode("closed"))
end

-- inboxTotal is how many messages of this inbox are steps of the run.
local function inboxTotal()
  local inboxPrefix = messagePrefix .. field .. ":"
  local total = 0
  for _, id in ipairs(redis.call("LRANGE", KEYS[4], 0, -1)) do
    if string.sub(id, 1, #inboxPrefix) == inboxPrefix then
      total = total + 1
    end
  end

  return total
end

-- flush saves the buffered messages as steps in arrival order, and empties the
-- buffer. It returns how many were saved and the limit that stopped it, if any.
local function flush(st, a)
  local saved = 0
  for _, entry in ipairs(buffer(st)) do
    local reason = saveMessage(entry.i, entry.v, a)
    if reason then
      closeForLimit(reason)

      return saved, reason
    end
    saved = saved + 1
  end

  redis.call("HDEL", KEYS[1], field)

  return saved, nil
end

local function limitArgs(from)
  return {ARGV[from], ARGV[from + 1], ARGV[from + 2], ARGV[from + 3]}
end

if redis.call("EXISTS", KEYS[3]) == 0 then
  return {"norun"}
end

if op == "deliver" then
  local st = load()
  if st == "closed" then
    return {"closed"}
  end

  local id, value, a = ARGV[3], ARGV[4], limitArgs(5)

  local w = waitingOp(st)
  if w then
    local reason = saveMessage(id, value, a)
    if reason then
      closeForLimit(reason)

      return {"limit", reason, w}
    end

    return {"wake", w}
  end

  local buf = buffer(st)
  if redis.call("HEXISTS", KEYS[2], id) == 1 then
    return {"dup"}
  end
  for _, entry in ipairs(buf) do
    if entry.i == id then
      return {"dup"}
    end
  end

  if #buf >= tonumber(a[1]) then
    local _, reason = flush(st, a)
    reason = reason or ("the subscription saved " .. a[1] .. " messages")
    closeForLimit(reason)

    return {"limit", reason, ""}
  end

  table.insert(buf, {i = id, v = value})
  redis.call("HSET", KEYS[1], field, cjson.encode(buf))

  return {"buffered"}
end

if op == "next" then
  local st = load()
  if st == "closed" then
    return {"closed"}
  end

  if #buffer(st) > 0 then
    local saved, reason = flush(st, limitArgs(4))
    if reason then
      return {"limit", reason}
    end

    return {"flushed", saved, inboxTotal()}
  end

  redis.call("HSET", KEYS[1], field, cjson.encode({waiting = ARGV[3]}))

  return {"waiting"}
end

if op == "flush" or op == "close" then
  local st = load()
  local saved, reason = 0, nil
  if st ~= "closed" then
    saved, reason = flush(st, limitArgs(3))
  end
  if op == "close" then
    redis.call("HSET", KEYS[1], field, cjson.encode("closed"))
  end
  if reason then
    return {"limit", reason, inboxTotal()}
  end

  return {"flushed", saved, inboxTotal()}
end

if op == "timeout" then
  local w = ARGV[3]
  if waitingOp(load()) == w then
    save(w, "null")
    redis.call("HDEL", KEYS[1], field)

    return {1}
  end

  -- A retry of a timer that already won finds the step it saved.
  if redis.call("HGET", KEYS[2], w) == "null" and redis.call("HEXISTS", KEYS[1], field) == 0 then
    return {1}
  end

  return {0}
end

if op == "clear" then
  if waitingOp(load()) == ARGV[3] then
    redis.call("HDEL", KEYS[1], field)
  end

  return {1}
end

return redis.error_reply("unknown subscription state op " .. tostring(op))
