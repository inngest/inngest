-- gets a decoded queue item
local function get_queue_item(queueKey, queueID)
	local fetched = redis.call("HGET", queueKey, queueID)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end
