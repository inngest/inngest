--[[

Closes a handle of an inbox. Saves the close op's result, takes the close op out
of the pending set, then ends the reads the op names. A read that is recorded
leaves the pending set and is returned with its record, which stays until the
caller has queued its wake. Every named read is marked closed, so a read that
is recorded after this call (the two ops can arrive in one response, in either
order) is ended as it is recorded.

Closing again saves nothing, and returns the reads that are still recorded, so
a close that crashed before its wakes were queued is repaired by running again.

Output:
  { pending, id, record, ... }  pending is 1 when pending steps remain, else 0;
                                then each recorded read's id and record

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

local stepID = ARGV[1]
local value = ARGV[2]

if redis.call("HEXISTS", keyStep, stepID) == 0 then
	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, stepID, value)
	redis.call("RPUSH", keyStack, stepID)
end

redis.call("SREM", keyStepsPending, stepID)

local ended = {}

for i = 3, #ARGV do
	local waitID = ARGV[i]
	local record = redis.call("HGET", keyWaits, waitID)

	if record then
		redis.call("SREM", keyStepsPending, waitID)

		table.insert(ended, waitID)
		table.insert(ended, record)
	end

	-- A decoded mark has no inbox, so an arrival passes over it.
	redis.call("HSET", keyWaits, "closed:" .. waitID, "{}")
end

local result = { redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
for _, v in ipairs(ended) do
	table.insert(result, v)
end

return result
