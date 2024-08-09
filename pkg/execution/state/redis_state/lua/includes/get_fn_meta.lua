-- gets a decoded function metadata hash
local function get_fn_meta(fnMetaKey)
	local fetched = redis.call("GET", fnMetaKey)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end
