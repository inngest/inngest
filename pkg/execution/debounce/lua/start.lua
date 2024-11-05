--[[
--  Updates the debounce pointer to something else
--  on function start so it doesn't rely on the existing
--  one on new events
-- ]]

local debouncePointerKey = KEYS[2]

local newDebounceID = ARGV[1]

redis.call("SET", debouncePointerKey, newDebounceID)

return "0"
