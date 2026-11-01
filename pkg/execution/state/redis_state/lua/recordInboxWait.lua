--[[

Records a wait on an inbox, then scans the steps saved after the step the wait
last saw. If one of them is a message of the inbox, the wait is satisfied: it
leaves the pending set but stays recorded until the caller has queued its wake.

The record is written before the scan, and an arrival saves before it looks at
the records, so at least one of the two always sees the other.

Output:
  { 0 }                           recorded; nothing for it to read
  { 1, messageStepID, pending }   recorded and satisfied; pending is 1 when
                                  pending steps remain, else 0

]]

local keyStep = KEYS[1]
local keyStack = KEYS[2]
local keyStepsPending = KEYS[3]
local keyWaits = KEYS[4]

local waitID = ARGV[1]
local record = ARGV[2]
local inbox = ARGV[3]
local seen = ARGV[4]

redis.call("HSET", keyWaits, waitID, record)

-- Walk back from the tail to the step the SDK last saw; those are the steps
-- saved since, and the only ones worth reading.
local gap = {}
for i = redis.call("LLEN", keyStack) - 1, 0, -1 do
	local stepID = redis.call("LINDEX", keyStack, i)

	if stepID == seen then
		break
	end

	table.insert(gap, 1, stepID)
end

for _, stepID in ipairs(gap) do
	local value = redis.call("HGET", keyStep, stepID)

	if value then
		local ok, decoded = pcall(cjson.decode, value)

		-- Only the executor writes this key, in the value of a message step.
		if ok and type(decoded) == "table" and decoded["inngest/inbox"] == inbox then
			redis.call("SREM", keyStepsPending, waitID)

			return { 1, stepID, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
		end
	end
end

return { 0 }
