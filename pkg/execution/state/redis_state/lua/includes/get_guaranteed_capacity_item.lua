local function get_guaranteed_capacity_item(keyGuaranteedCapacityMap, guaranteedCapacityName)
    local fetched = redis.call("HGET", keyGuaranteedCapacityMap, guaranteedCapacityName)
    if fetched ~= false then
        return cjson.decode(fetched)
    end
    return nil
end
