--[[

Saves a message step of an inbox, then finds the recorded waits on that inbox
that the message satisfies. Each satisfied wait leaves the pending set; it stays
recorded until the caller has queued its wake.

Saving the same step with the same value again saves nothing but still finds the
waits, so a redelivered event repairs a wake that was lost.

Output:
  { -1 }                         different data is already saved under the step
  { 0|-2, id, record, pending... }  saved (0) or already saved (-2), then for each
                                  satisfied wait its id, its record and whether
                                  pending steps remain (1) or not (0)

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

local stepID = ARGV[1]
local value = ARGV[2]
local inbox = ARGV[3]

local status = 0

if redis.call("HEXISTS", keyStep, stepID) == 1 then
	if redis.call("HGET", keyStep, stepID) ~= value then
		return { -1 }
	end

	status = -2
else
	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, stepID, value)
	redis.call("RPUSH", keyStack, stepID)
end

local result = { status }

local waits = redis.call("HGETALL", keyWaits)
for i = 1, #waits, 2 do
	local wait = cjson.decode(waits[i + 1])

	if wait.inbox == inbox then
		redis.call("SREM", keyStepsPending, waits[i])

		result[#result + 1] = waits[i]
		result[#result + 1] = waits[i + 1]
		result[#result + 1] = redis.call("SCARD", keyStepsPending) > 0 and 1 or 0
	end
end

return result
