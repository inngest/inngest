local function get_request_lease_item(keyRequestLease)
	local fetched = redis.call("GET", keyRequestLease)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end
