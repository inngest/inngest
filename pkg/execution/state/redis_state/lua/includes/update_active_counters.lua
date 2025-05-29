local function increaseActiveCounters(keyActivePartition, keyActiveAccount, keyActiveCompound, keyActiveConcurrencyKey1, keyActiveConcurrencyKey2, refilled)
  -- Increase active counters by number of refilled items
  redis.call("INCRBY", keyActivePartition, refilled)

  if exists_without_ending(keyActiveAccount, ":-") then
    redis.call("INCRBY", keyActiveAccount, refilled)
  end

  if exists_without_ending(keyActiveCompound, ":-") then
    redis.call("INCRBY", keyActiveCompound, refilled)
  end

  if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
    redis.call("INCRBY", keyActiveConcurrencyKey1, refilled)
  end

  if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
    redis.call("INCRBY", keyActiveConcurrencyKey2, refilled)
  end
end

local function decreaseActiveCounters(keyActivePartition, keyActiveAccount, keyActiveCompound, keyActiveConcurrencyKey1, keyActiveConcurrencyKey2)
  -- Decrease active counters and clean up if necessary
  if redis.call("DECR", keyActivePartition) <= 0 then
    redis.call("DEL", keyActivePartition)
  end

  if exists_without_ending(keyActiveAccount, ":-") then
    if redis.call("DECR", keyActiveAccount) <= 0 then
      redis.call("DEL", keyActiveAccount)
    end
  end

  if exists_without_ending(keyActiveCompound, ":-") then
    if redis.call("DECR", keyActiveCompound) <= 0 then
      redis.call("DEL", keyActiveCompound)
    end
  end

  if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
    if redis.call("DECR", keyActiveConcurrencyKey1) <= 0 then
      redis.call("DEL", keyActiveConcurrencyKey1)
    end
  end

  if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
    if redis.call("DECR", keyActiveConcurrencyKey2) <= 0 then
      redis.call("DEL", keyActiveConcurrencyKey2)
    end
  end
end

local function increaseActiveRunCounters(keyActiveRun, keyIndexActivePartitionRuns, keyActiveRunsAccount, keyActiveRunsCustomConcurrencyKey1, keyActiveRunsCustomConcurrencyKey2, runID)
  -- increase number of active items in run
  if redis.call("INCR", keyActiveRun) == 1 then
    -- if the first item in a run was moved to the ready queue, mark the run as active
    -- and increment counters
    if exists_without_ending(keyIndexActivePartitionRuns, ":-") then
      redis.call("SADD", keyIndexActivePartitionRuns, runID)
    end

    if exists_without_ending(keyActiveRunsAccount, ":-") then
      redis.call("INCR", keyActiveRunsAccount)
    end

    if exists_without_ending(keyActiveRunsCustomConcurrencyKey1, ":-") then
      redis.call("INCR", keyActiveRunsCustomConcurrencyKey1)
    end

    if exists_without_ending(keyActiveRunsCustomConcurrencyKey2, ":-") then
      redis.call("INCR", keyActiveRunsCustomConcurrencyKey2)
    end
  end
end

local function decreaseActiveRunCounters(keyActiveRun, keyIndexActivePartitionRuns, keyActiveRunsAccount, keyActiveRunsCustomConcurrencyKey1, keyActiveRunsCustomConcurrencyKey2, runID)
  if exists_without_ending(keyActiveRun, ":-") then
    -- increase number of active items in the run
    if redis.call("DECR", keyActiveRun) <= 0 then
      redis.call("DEL", keyActiveRun)

      -- update set of active function runs
      if exists_without_ending(keyIndexActivePartitionRuns, ":-") then
        redis.call("SREM", keyIndexActivePartitionRuns, runID)
      end

      if exists_without_ending(keyActiveRunsAccount, ":-") then
        if redis.call("DECR", keyActiveRunsAccount) <= 0 then
          redis.call("DEL", keyActiveRunsAccount)
        end
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey1, ":-") then
        if redis.call("DECR", keyActiveRunsCustomConcurrencyKey1) <= 0 then
          redis.call("DEL", keyActiveRunsCustomConcurrencyKey1)
        end
      end

      if exists_without_ending(keyActiveRunsCustomConcurrencyKey2, ":-") then
        if redis.call("DECR", keyActiveRunsCustomConcurrencyKey2) <= 0 then
          redis.call("DEL", keyActiveRunsCustomConcurrencyKey2)
        end
      end
    end
  end
end
