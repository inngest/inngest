--[[

  Sets the "migrate" value on the metadata for the given function metadata key.

  Return values:
  0 - Updated successfully

]]

local keyFnMeta = KEYS[1]

local migrate     = tonumber(ARGV[1])
local defaultMeta  = ARGV[2]

-- $include(get_fn_meta.lua)
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
