--
-- Helper functions
--

-- Check if value is empty
local function is_empty(val)
  return val == nil or val == "" or val == false
end

-- Update the pointer ULID to a new value
local function update_pointer(key, id)
  redis.call("SET", key, id)
end

-- Check if a field in the Map exists or not
local function is_meta_empty(key, field)
  return redis.call("HEXISTS", key, field) == 0
end

local function is_status_empty(key)
  return is_meta_empty(key, "status")
end

local function set_batch_status(key, value)
  redis.call("HSET", key, "status", value)
end

local function get_batch_status(key)
  return redis.call("HGET", key, "status")
end

-- Legacy and mixed-writer batches keep their original size check until deleted.
-- Only new batches, with no unaccounted-for appends, use the serialized tally.
local function update_batch_bytes(batchKey, metadataKey, previousCount, count, addedBytes)
  local bytes = 0
  if previousCount > 0 then
    -- TODO: Remove bytes_count and this fallback once all writers are upgraded and old batches have drained.
    local tally = redis.call("HMGET", metadataKey, "bytes", "bytes_count")
    if is_empty(tally[1]) or tonumber(tally[2]) ~= previousCount then
      return redis.call("MEMORY", "USAGE", batchKey) or 0
    end
    bytes = tonumber(tally[1])
  end

  bytes = bytes + addedBytes
  redis.call("HSET", metadataKey, "bytes", bytes, "bytes_count", count)
  return bytes
end
