-- This function updates account queues
-- Requires: update_pointer_score.lua, ends_with.lua
local function update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountId, score)
  -- we might be leasing an "old" partition which doesn't store the account
  if account_is_set(keyAccountPartitions) == true then
    update_pointer_score_to(partitionID, keyAccountPartitions, score)

    -- Upsert global accounts to _earliest_ score
    local earliestPartitionScoreInAccount = get_fn_partition_score(keyAccountPartitions)
    update_pointer_score_to(accountId, keyGlobalAccountPointer, earliestPartitionScoreInAccount)
  end
end

local function account_is_set(keyAccountPartitions)
  return exists_without_ending(keyAccountPartitions, "accounts:00000000-0000-0000-0000-000000000000:partition:sorted") == true
end
