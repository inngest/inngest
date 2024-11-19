--[[

  Upsert connection to map

]]

local connKey = KEYS[1]
local groupKey = KEYS[2]
local groupIDKey = KEYS[3]

local connID = ARGV[1]
local connMeta = ARGV[2]
local groupID = ARGV[3]
local workerGroup = ARGV[4]

-- Store the connection metadata in a map
redis.call("HSET", connKey, connID, connMeta)

-- Store the group if it doesn't exist yet
redis.call("HSETNX", groupKey, groupID, workerGroup)

-- Add connID into the group
redis.call("SADD", groupIDKey, connID)

return 0
