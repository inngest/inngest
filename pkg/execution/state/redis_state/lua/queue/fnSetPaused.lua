--[[

  Sets the "off" boolean on the metadata for the given function metadata key.

  Return values:
  0 - Updated "off" boolean

]]

local keyFnMeta = KEYS[1]

local isPaused     = tonumber(ARGV[1])
local defaultMeta  = ARGV[2]

-- $include(get_fn_meta.lua)
local existing = get_fn_meta(keyFnMeta)
if existing == nil then
	redis.call("SET", keyFnMeta, defaultMeta)
	return 0
end

if isPaused == 1 then
    existing.off = true
else
    existing.off = false
end
redis.call("SET", keyFnMeta, cjson.encode(existing))

return 0
