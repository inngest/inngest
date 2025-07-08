local function updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)
  -- Retrieve the earliest item score in the backlog in milliseconds
  local earliestBacklogScore = get_earliest_score(keyBacklogSet)

  -- If backlog is empty, update dangling pointers in shadow partition
  if earliestBacklogScore == 0 then
    -- Remove meta
    redis.call("HDEL", keyBacklogMeta, backlogID)

    redis.call("ZREM", keyShadowPartitionSet, backlogID)

    -- If shadow partition has no more backlogs, update global/account pointers
    if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
      -- Remove meta, only if no more async normalizations are due
      if tonumber(redis.call("ZCARD", keyPartitionNormalizeSet)) == 0 then
        redis.call("HDEL", keyShadowPartitionMeta, partitionID)
      end

      redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
      redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

      if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
        redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
      end
    end

    return
  end

  -- If backlog has more items, update pointer in shadow partition
  update_pointer_score_to(backlogID, keyShadowPartitionSet, earliestBacklogScore)

  -- In case the backlog is the new earliest item in the shadow partition,
  -- update pointers to shadow partition in global indexes
  local earliestShadowPartitionScore = get_earliest_score(keyShadowPartitionSet)

  -- Push back shadow partition in global set
  update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, earliestShadowPartitionScore)

  -- Push back shadow partition in account set + potentially push back account in global accounts set
  update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, earliestShadowPartitionScore)
end
