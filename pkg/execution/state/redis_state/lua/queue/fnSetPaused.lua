--[[

  Sets the "off" boolean on the metadata for the given function metadata key.

  Return values:
  0 - Updated "off" boolean

]]

local keyFnMeta               = KEYS[1]
local keyShadowPartitionMeta  = KEYS[2]

local isPaused     = tonumber(ARGV[1])
local defaultMeta  = ARGV[2]
local partitionID  = ARGV[3]

-- $include(get_fn_meta.lua)
-- $include(get_partition_item.lua)

local existingMeta = get_fn_meta(keyFnMeta)

-- update shadow partition
local existingShadow = get_shadow_partition_item(keyShadowPartitionMeta, partitionID)
if existingShadow ~= nil and existingShadow ~= false then
  if isPaused == 1 or existingMeta.migrate == true then
    existingShadow.norefill = true
  else
    existingShadow.norefill = false
  end
  redis.call("HSET", keyShadowPartitionMeta, partitionID, cjson.encode(existingShadow))
end

if existingMeta == nil then
	redis.call("SET", keyFnMeta, defaultMeta)
	return 0
end

if isPaused == 1 then
    existingMeta.off = true
else
    existingMeta.off = false
end
redis.call("SET", keyFnMeta, cjson.encode(existingMeta))

return 0
