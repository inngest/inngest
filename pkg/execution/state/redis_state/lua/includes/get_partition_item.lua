-- gets a decoded partition item
local function get_partition_item(partitionKey, id)
	local fetched = redis.call("HGET", partitionKey, id)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end
