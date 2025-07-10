local keyBacklogActiveCheckSet   = KEYS[1]
local keyBacklogActiveCheckCooldown = KEYS[2]

local backlogID       = ARGV[1]
local nowMS           = tonumber(ARGV[2])

-- $include(active_check.lua)

add_to_active_check(keyBacklogActiveCheckSet, keyBacklogActiveCheckCooldown, backlogID, nowMS)

return 0
