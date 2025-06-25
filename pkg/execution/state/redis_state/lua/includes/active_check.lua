local function add_to_active_check(keyBacklogActiveCheckSet, keyBacklogActiveCheckCooldown, backlogID, nowMS)
  if redis.call("EXISTS", keyBacklogActiveCheckCooldown) == 1 then
    return
  end

  -- Protect against overflowing the active check set -- this should be a best effort workload
  if tonumber(redis.call("ZCARD", keyBacklogActiveCheckSet)) >= 1000 then
    return
  end

  redis.call("ZADD", keyBacklogActiveCheckSet, nowMS, backlogID)
end
