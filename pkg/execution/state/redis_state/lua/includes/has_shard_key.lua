local function has_shard_key(key)
	return string.sub(key, -2) ~= ":-"
end
