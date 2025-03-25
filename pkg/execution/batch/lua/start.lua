--
-- Check if the batch has already started or not.
--
-- Return values:
--   0: Can start
--   1: Already started
--
local batchMetadataKey = KEYS[1] -- key for batch metadata
local batchPointerKey = KEYS[2]  -- key for pointer

local batchStatusStarted = ARGV[1]
local newBatchID = ARGV[2] -- the ULID for a new batch

-- $include(helpers.lua)

local status = get_batch_status(batchMetadataKey)

-- return if already started
if status == batchStatusStarted then
  return 1
end

update_pointer(batchPointerKey, newBatchID)

if is_status_empty(batchMetadataKey) then
  -- status doesn't exist, something is wrong, abort
  return -1
end

set_batch_status(batchMetadataKey, batchStatusStarted)

return 0
