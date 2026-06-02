--
-- Check if the batch has already started or not.
--
-- Return values:
--   0: Can start
--   1: Already started
--  -1: Batch metadata absent
--
local batchMetadataKey = KEYS[1] -- key for batch metadata
local batchPointerKey = KEYS[2]  -- key for pointer

local batchStatusStarted = ARGV[1]
local newBatchID = ARGV[2] -- the ULID for a new batch
local batchID = ARGV[3]    -- the batch ID being started (optional, for conditional pointer update)

-- $include(helpers.lua)

local status = get_batch_status(batchMetadataKey)

-- return if already started
if status == batchStatusStarted then
  return 1
end

-- Abort before writing any state if batch metadata doesn't exist.
-- This must be checked before the pointer update to avoid corrupting
-- state when the batch lives on a different cluster during migration.
if is_status_empty(batchMetadataKey) then
  return -1
end

-- Only update the pointer if it currently points to this batch.
-- This prevents overwriting the pointer when bulk_append has already
-- created an overflow batch and updated the pointer to it.
local currentPointer = redis.call("GET", batchPointerKey)
if is_empty(batchID) or is_empty(currentPointer) or currentPointer == batchID then
  update_pointer(batchPointerKey, newBatchID)
end

set_batch_status(batchMetadataKey, batchStatusStarted)

return 0
