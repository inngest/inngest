local function add_to_active_check(keyPartitionActiveCheckSet, keyPartitionActiveCheckCooldown, partitionID, nowMS)
  if redis.call("EXISTS", keyPartitionActiveCheckCooldown) == 1 then
    return
  end

  -- Protect against overflowing the active check set -- this should be a best effort workload
  if tonumber(redis.call("ZCARD", keyPartitionActiveCheckSet)) >= 1000 then
    return
  end

  redis.call("ZADD", keyPartitionActiveCheckSet, nowMS, partitionID)
end
