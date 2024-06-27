--[[

  Sets the "off" boolean on the metadata for the given function metadata key.

  Return values:
  0 - Updated "off" boolean
  1 - Function meta not found

]]

local fnMetaKey = KEYS[1]

local isPaused     = tonumber(ARGV[1])

-- $include(get_fn_meta.lua)
local existing = get_fn_meta(fnMetaKey)
if existing == nil then
	return 1
end

if isPaused == 1 then
    existing.off = true
else
    existing.off = false
end
redis.call("SET", fnMetaKey, cjson.encode(existing))

return 0
