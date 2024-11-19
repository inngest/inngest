--[[

  Delete the connection

  Return:
    0: conn is deleted successfully
    1: group is deleted as well
]]

local connKey = KEYS[1]
local groupKey = KEYS[2]
local groupIDKey = KEYS[3]
local indexConnectionsByAppIdKey = KEYS[4]

local connID = ARGV[1]
local groupID = ARGV[2]

-- $include(ends_with.lua)

-- Remove the connection from the map
redis.call("HDEL", connKey, connID)

-- Remove the connID from the group set, set is deleted when empty
redis.call("SREM", groupIDKey, connID)

if exists_without_ending(indexConnectionsByAppIdKey, "index_disabled") then
	redis.call("SREM", indexConnectionsByAppIdKey, connID)
end

-- If the group is empty, remove it
local scount = tonumber(redis.call("SCARD", groupIDKey))
if scount == 0 then
  redis.call("HDEL", groupKey, groupID)

  return 1
end

return 0
