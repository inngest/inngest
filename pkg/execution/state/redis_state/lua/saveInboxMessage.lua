--[[

Saves a message step of an inbox, then finds the recorded waits on that inbox
that want the message. Each satisfied wait leaves the pending set; it stays
recorded until the caller has queued its wake.

Saving the same step with the same value again saves nothing but still finds the
waits, so a redelivered event repairs a wake that was lost.

A new message that would take the run past its step bound or its state size
bound is not saved. A bound of 0 means the bound is not checked.

A satisfied batch read is also answered: the ids of the messages it wants saved
since the step it last saw are saved under its id, unless it already has a result.

Output:
  { -1 }                         different data is already saved under the step
  { -3 }                         the run is at its step bound; nothing was saved
  { -4 }                         the message would pass the state size bound;
                                 nothing was saved
  { 0|-2, id, record, pending... }  saved (0) or already saved (-2), then for each
                                  satisfied wait its id, its record and whether
                                  pending steps remain (1) or not (0)

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

-- $include(inbox_wants.lua)
-- $include(inbox_batch.lua)

local stepID = ARGV[1]
local value = ARGV[2]
local inbox = ARGV[3]
local maxSteps = tonumber(ARGV[4])
local maxState = tonumber(ARGV[5])

local status = 0

if redis.call("HEXISTS", keyStep, stepID) == 1 then
	if redis.call("HGET", keyStep, stepID) ~= value then
		return { -1 }
	end

	status = -2
else
	if maxSteps > 0 and tonumber(redis.call("HGET", keyMetadata, "step_count") or "0") >= maxSteps then
		return { -3 }
	end

	if maxState > 0 and tonumber(redis.call("HGET", keyMetadata, "state_size") or "0") + #value > maxState then
		return { -4 }
	end

	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, stepID, value)
	redis.call("RPUSH", keyStack, stepID)
end

local result = { status }
local message = cjson.decode(value)

local waits = redis.call("HGETALL", keyWaits)
for i = 1, #waits, 2 do
	local wait = cjson.decode(waits[i + 1])

	if wait.inbox == inbox and inbox_wants(wait.want, message) then
		-- A batch read is answered with every message it wants saved since it
		-- last saw the stack, this one included.
		if wait.batch ~= nil then
			local ids = inbox_batch_ids(keyStep, keyStack, inbox, wait.want, wait.seen, wait.batch.room)

			inbox_batch_save(keyStep, keyMetadata, keyStack, waits[i], wait.seen, ids)
		end

		redis.call("SREM", keyStepsPending, waits[i])

		result[#result + 1] = waits[i]
		result[#result + 1] = waits[i + 1]
		result[#result + 1] = redis.call("SCARD", keyStepsPending) > 0 and 1 or 0
	end
end

return result
