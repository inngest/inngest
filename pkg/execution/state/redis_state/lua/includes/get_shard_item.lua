local function get_shard_item(keyShardMap, shardName)
    local fetched = redis.call("HGET", keyShardMap, shardName)
    if fetched ~= false then
        return cjson.decode(fetched)
    end
    return nil
end
