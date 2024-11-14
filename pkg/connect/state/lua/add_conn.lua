--[[

  Add connection to map

]]

local connKey = KEYS[1]
local groupKey = KEYS[2]

local connID = ARGV[1]
local connMeta = ARGV[2]

-- Store the connection metadata in a map
redis.call("HSET", connKey, connID, connMeta)

-- Add connID into the group
redis.call("SADD", groupKey, connID)


return 0
