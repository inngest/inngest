--
--This script runs batch append ops in an atomic action
--

local batchPointerKey = KEYS[1]      -- key to the batch pointer

local batchLimit = tonumber(ARGV[1]) -- max size configured for this batch
local event = ARGV[2]                -- event to be appended to the batch
local newULID = ARGV[3]              -- ULID to update the pointer with, either if the batch is full or doesn't exist
local prefix = ARGV[4]               -- the prefix used for redis

local batchStatusAppending = ARGV[5]
local batchStatusStarted = ARGV[6]
local batchSizeLimit = tonumber(ARGV[7])

-- helper functions
-- $include(helpers.lua)

local function get_or_create_batch_key(key)
  local val = redis.call("GET", key)

  -- if empty or doesn't exist
  if is_empty(val) then
    -- create new pointer by setting the ULID
    update_pointer(batchPointerKey, newULID)
    val = newULID
  end

  return val
end

-- start execution
local batchID = get_or_create_batch_key(batchPointerKey)
local resp = { status = "append", batchID = batchID, batchPointerKey = batchPointerKey }

-- NOTE: these need to be identical to the ones in the queue key generator
--   * Batch
--   * BatchMetadata
local keyfmt = "%s:batches:%s"
local batchKey = string.format(keyfmt, prefix, batchID)
local batchMetadataKey = string.format("%s:metadata", batchKey)

-- set the batch status if it doesn't exist but don't overwrite
-- this is necessary for functions that never enabled batch before
if is_status_empty(batchMetadataKey) then
  set_batch_status(batchMetadataKey, batchStatusAppending)
end

-- append event to batch
local len = redis.call("RPUSH", batchKey, event)

if len == 1 then
    -- newly started batch
    resp = { status = "new", batchID = batchID, batchPointerKey = batchPointerKey }
end

local size = redis.call("MEMORY", "USAGE", batchKey)
-- if batch is full
if len >= batchLimit or size >= batchSizeLimit then
  if not is_status_empty(batchMetadataKey) then
    set_batch_status(batchMetadataKey, batchStatusStarted)
  end

  -- change poiner so following ops don't append to this batch anymore
  update_pointer(batchPointerKey, newULID)

  -- Check if the batch size limit is reached, this inevitably will go over a little bit
  -- but that should be fine consider there's a cap on the size of an event
  local status = "full"
  if size >= batchSizeLimit then
    status = "maxsize"
  end

  resp = { status = status, batchID = batchID, batchPointerKey = batchPointerKey }
end

return cjson.encode(resp)
