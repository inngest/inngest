--[[
--  Updates the debounce pointer to something else
--  on function start so it doesn't rely on the existing
--  one on new events
--
--  Return value:
--   -1: migrating
--    0: untouched
--    1: updated
-- ]]

local debouncePointerKey = KEYS[1]
local keyDebounceMigrating = KEYS[2]

local newDebounceID = ARGV[1]
local existingDebounceID = ARGV[2]

local currentID = redis.call("GET", debouncePointerKey)

-- If debounce is being migrated, we don't want to run the timeout job.
if redis.call("HEXISTS", keyDebounceMigrating, existingDebounceID) == 1 then
	return -1
end

-- In case we are racing with prepareMigration() and get here first, we will update the pointer to a new debounce ID,
-- so the migration will not find any debounces.

-- update the pointer key value only if the existing one matches
if currentID ~= nil and currentID ~= false and currentID == existingDebounceID then
  redis.call("SET", debouncePointerKey, newDebounceID)
  return 1
end

return 0
