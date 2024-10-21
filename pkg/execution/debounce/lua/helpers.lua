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
