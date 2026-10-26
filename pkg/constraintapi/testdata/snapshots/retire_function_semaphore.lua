local prefix, functionID, mode = ARGV[1], ARGV[2], ARGV[3]
local limit = tonumber(ARGV[4])
if redis.call("ZCARD", KEYS[1]) > limit then
  return redis.error_reply("ERR account lease inventory exceeds maintenance bound")
end
local leases = redis.call("ZRANGE", KEYS[1], 0, limit)
for _, lease in ipairs(leases) do
  local requestID = redis.call("HGET", prefix .. ":ld:" .. lease, "req")
  if not requestID or requestID == "" then
    return redis.error_reply("ERR lease details are missing")
  end
  local raw = redis.call("GET", prefix .. ":rs:" .. requestID)
  if not raw then return redis.error_reply("ERR lease request is missing") end
  local ok, request = pcall(cjson.decode, raw)
  if not ok or type(request) ~= "table" or type(request.f) ~= "string" then
    return redis.error_reply("ERR lease request identity is unknown")
  end
  if request.f == functionID then
    return redis.error_reply("ERR function still has releasable capacity leases")
  end
  if mode == "retire" then
    if type(request.s) ~= "table" then return redis.error_reply("ERR lease constraints are unknown") end
    for _, constraint in ipairs(request.s) do
      if constraint.k == 4 then
        if type(constraint.sem) ~= "table" or type(constraint.sem.k) ~= "string" then
          return redis.error_reply("ERR lease semaphore is unknown")
        end
        if constraint.sem.k == KEYS[2] then
          return redis.error_reply("ERR another function still references the semaphore bucket")
        end
      end
    end
  end
end
if mode == "retire" then
  redis.call("DEL", KEYS[2])
end
return 1