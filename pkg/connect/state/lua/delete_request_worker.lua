--[[

Deletes the request-to-worker key only when it names the given worker instance.

Output:
  0: Key is missing or names a different instance, nothing deleted
  1: Key deleted

ARGV[1]: Instance ID that is requesting the deletion
]]

local requestWorkerKey = KEYS[1]
local instanceID = ARGV[1]

if redis.call("GET", requestWorkerKey) ~= instanceID then
  return 0
end

return redis.call("DEL", requestWorkerKey)
