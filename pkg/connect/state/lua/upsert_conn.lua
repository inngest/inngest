--[[

  Upsert connection to map

]]

local connKey = KEYS[1]
local groupKey = KEYS[2]
local groupIDKey = KEYS[3]
local indexConnectionsByAppIdKey = KEYS[4]

local connID = ARGV[1]
local connMeta = ARGV[2]
local groupID = ARGV[3]
local workerGroup = ARGV[4]
local isHealthy = tonumber(ARGV[5])

-- $include(ends_with.lua)

-- Store the connection metadata in a map
redis.call("HSET", connKey, connID, connMeta)

-- Store the group if it doesn't exist yet
redis.call("HSETNX", groupKey, groupID, workerGroup)

-- Add connID into the group
redis.call("SADD", groupIDKey, connID)

if exists_without_ending(indexConnectionsByAppIdKey, "index_disabled") then
	if isHealthy == 1 then
		redis.call("SADD", indexConnectionsByAppIdKey, connID)
	else
		redis.call("SREM", indexConnectionsByAppIdKey, connID)
	end
end

return 0
