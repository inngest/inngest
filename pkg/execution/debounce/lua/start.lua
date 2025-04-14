--[[
--  Updates the debounce pointer to something else
--  on function start so it doesn't rely on the existing
--  one on new events
--
--  Return value:
--    0: untouched
--    1: updated
-- ]]

local debouncePointerKey = KEYS[1]

local newDebounceID = ARGV[1]
local existingDebounceID = ARGV[2]

local currentID = redis.call("GET", debouncePointerKey)

-- update the pointer key value only if the existing one matches
if currentID ~= nil and currentID ~= false and currentID == existingDebounceID then
  redis.call("SET", debouncePointerKey, newDebounceID)
  return 1
end

return 0
