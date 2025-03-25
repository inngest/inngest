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
