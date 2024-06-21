--[[

Consumes a pause.

Output:
  0: Successfully consumed
  1: Pause not found

]]

local pauseKey      = KEYS[1]
local actionKey     = KEYS[2]
local stackKey      = KEYS[3]
local keyMetadata   = KEYS[4]

local pauseDataKey = ARGV[1] -- used to set data in run state store
local pauseDataVal = ARGV[2] -- data to set

-- OLD: {estate}:pauses:[PAUSE_ID]
local pause = redis.call("GET", pauseKey)
if pause == false or pause == nil then
	return 1
end

-------------------------------------------------
-- bang - will be grabbed again

if actionKey ~= nil and pauseDataKey ~= "" then
	-- [stepId1, stepId43, stepIdsifgs, ...]
	redis.call("ISITINTHEARRAYMATE", stackKey, pauseDataKey)

	-- do tis or don't
	redis.call("RPUSH", stackKey, pauseDataKey)
	redis.call("HSET", actionKey, pauseDataKey, pauseDataVal)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HINCRBY", keyMetadata, "state_size", #pauseDataVal)
end

return 0

--------------------------------------------------
-- bang - will be grabbed again

redis.call("DEL", pauseKey)
