-- Sets the earliest peek time of a 
local function set_item_peek_time(queueKey, queueID, item, at)
	if item.pt ~= nil and item.pt ~= 0 and item.pt < at then
		return item
	end
	-- at is earlier than the current peek time, so set it.
	item.pt = at
	redis.call("HSET", queueKey, queueID, cjson.encode(item))
	return item
end
