local function addToActiveSets(keyActivePartition, keyActiveAccount, keyActiveCompound, keyActiveConcurrencyKey1, keyActiveConcurrencyKey2, itemIDs)
  -- Increase active sets by number of refilled items
  redis.call("SADD", keyActivePartition, unpack(itemIDs))

  if exists_without_ending(keyActiveAccount, ":-") then
    redis.call("SADD", keyActiveAccount, unpack(itemIDs))
  end

  if exists_without_ending(keyActiveCompound, ":-") then
    redis.call("SADD", keyActiveCompound, unpack(itemIDs))
  end

  if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
    redis.call("SADD", keyActiveConcurrencyKey1, unpack(itemIDs))
  end

  if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
    redis.call("SADD", keyActiveConcurrencyKey2, unpack(itemIDs))
  end
end

local function removeFromActiveSets(keyActivePartition, keyActiveAccount, keyActiveCompound, keyActiveConcurrencyKey1, keyActiveConcurrencyKey2, itemID)
  -- Decrease active sets and clean up if necessary
  redis.call("SREM", keyActivePartition, itemID)

  if exists_without_ending(keyActiveAccount, ":-") then
    redis.call("SREM", keyActiveAccount, itemID)
  end

  if exists_without_ending(keyActiveCompound, ":-") then
    redis.call("SREM", keyActiveCompound, itemID)
  end

  if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
    redis.call("SREM", keyActiveConcurrencyKey1, itemID)
  end

  if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
    redis.call("SREM", keyActiveConcurrencyKey2, itemID)
  end
end

local function addToActiveRunSets(keyActiveRun, keyActiveRunsPartition, keyActiveRunsAccount, keyActiveRunsCustomConcurrencyKey1, keyActiveRunsCustomConcurrencyKey2, runID, itemID)
  if exists_without_ending(keyActiveRun, ":-") then
    redis.call("SADD", keyActiveRun, itemID)

    -- if the first item in a run was moved to the ready queue, mark the run as active
    -- Note: While SADD is idempotent, we can reduce operations on Redis if we do a single SCARD first
    if tonumber(redis.call("SCARD", keyActiveRun)) == 1 then
      if exists_without_ending(keyActiveRunsPartition, ":-") then
        redis.call("SADD", keyActiveRunsPartition, runID)
      end

      if exists_without_ending(keyActiveRunsAccount, ":-") then
        redis.call("SADD", keyActiveRunsAccount, runID)
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey1, ":-") then
        redis.call("SADD", keyActiveRunsCustomConcurrencyKey1, runID)
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey2, ":-") then
        redis.call("SADD", keyActiveRunsCustomConcurrencyKey2, runID)
      end
    end
  end
end

local function removeFromActiveRunSets(keyActiveRun, keyActiveRunsPartition, keyActiveRunsAccount, keyActiveRunsCustomConcurrencyKey1, keyActiveRunsCustomConcurrencyKey2, runID, itemID)
  if exists_without_ending(keyActiveRun, ":-") then
    redis.call("SREM", keyActiveRun, itemID)

    -- if the last item was removed, remove the run from active
    if tonumber(redis.call("SCARD", keyActiveRun)) == 0 then
      if exists_without_ending(keyActiveRunsPartition, ":-") then
        redis.call("SREM", keyActiveRunsPartition, runID)
      end

      if exists_without_ending(keyActiveRunsAccount, ":-") then
        redis.call("SREM", keyActiveRunsAccount, runID)
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey1, ":-") then
        redis.call("SREM", keyActiveRunsCustomConcurrencyKey1, runID)
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey2, ":-") then
        redis.call("SREM", keyActiveRunsCustomConcurrencyKey2, runID)
      end
    end
  end
end
