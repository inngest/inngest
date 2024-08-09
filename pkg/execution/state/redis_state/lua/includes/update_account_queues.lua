local function account_is_set(keyAccountPartitions)
  return exists_without_ending(keyAccountPartitions, "accounts:00000000-0000-0000-0000-000000000000:partition:sorted") == true
end

-- get_fn_partition_score returns a fn's earliest job as a score for pointer queues.
-- This returns 0 if there are no scores available.
local function get_earliest_account_partition_score(keyAccountPartitions)
    local earliestScore = redis.call("ZRANGE", keyAccountPartitions, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
    if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
        return 0
    end
    -- pointers are already second precision, so we do not want to truncate any further (as opposed to get_fn_partition_score)
    -- earliest is a table containing {item, score}
    return tonumber(earliestScore[2])
end

-- This function updates account queues
-- Requires: update_pointer_score.lua, ends_with.lua
local function update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountId, score)
  -- we might be leasing an "old" partition which doesn't store the account
  if account_is_set(keyAccountPartitions) == true then
    update_pointer_score_to(partitionID, keyAccountPartitions, score)

    -- Upsert global accounts to _earliest_ score
    local earliestPartitionScoreInAccount = get_earliest_account_partition_score(keyAccountPartitions)
    update_pointer_score_to(accountId, keyGlobalAccountPointer, earliestPartitionScoreInAccount)
  end
end

