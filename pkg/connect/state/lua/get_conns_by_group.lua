--[[
  Retrive connections by groupID
]]

local connKey = KEYS[1]
local groupIDKey = KEYS[2]

-- retreive the list of connection IDs in the group
local connIDs = redis.call("SMEMBERS", groupIDKey)

if #connIDs == 0 then
  return {}
end

return redis.call("HMGET", connKey, unpack(connIDs))
