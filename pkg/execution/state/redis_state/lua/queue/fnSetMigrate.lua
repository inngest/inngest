--[[

  Sets the "migrate" value on the metadata for the given function metadata key.

  Return values:
  0 - Updated successfully

]]

local keyFnMeta               = KEYS[1]
local keyShadowPartitionMeta  = KEYS[2]

local migrate       = tonumber(ARGV[1])
local defaultMeta   = ARGV[2]
local partitionID   = ARGV[3]

-- $include(get_fn_meta.lua)
-- $include(get_partition_item.lua)

-- update shadow partition
local existing = get_shadow_partition_item(keyShadowPartitionMeta, partitionID)
if existing ~= nil and existing ~= false then
  if migrate == 1 then
    existing.norefill = true
  else
    existing.norefill = false
  end
  redis.call("HSET", keyShadowPartitionMeta, partitionID, cjson.encode(existing))
end

local existing = get_fn_meta(keyFnMeta)
if existing == nil then
	redis.call("SET", keyFnMeta, defaultMeta)
	return 0
end

if migrate >= 1 then
  existing.migrate = true
else
  existing.migrate = false
end
redis.call("SET", keyFnMeta, cjson.encode(existing))

return 0
