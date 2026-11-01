--[[

Records a wait on an inbox, then scans the steps saved after the step the wait
last saw. If one of them is a message of the inbox that the wait wants, the wait
is satisfied: it leaves the pending set but stays recorded until the caller has
queued its wake.

The record is written before the scan, and an arrival saves before it looks at
the records, so at least one of the two always sees the other.

A read whose handle was closed (the close marked it) is not recorded: it
leaves the pending set and is ended.

Output:
  { 0 }                           recorded; nothing for it to read
  { 1, messageStepID, pending }   recorded and satisfied; pending is 1 when
                                  pending steps remain, else 0
  { 2, pending }                  not recorded; its handle was closed
  { 3, pending }                  a batch read, recorded and answered with the
                                  messages saved since it last saw the stack

]]

local keyStep = KEYS[1]
local keyStack = KEYS[2]
local keyStepsPending = KEYS[3]
local keyWaits = KEYS[4]
local keyMetadata = KEYS[5]

-- $include(inbox_wants.lua)
-- $include(inbox_batch.lua)

local waitID = ARGV[1]
local record = ARGV[2]
local inbox = ARGV[3]
local seen = ARGV[4]

local decodedRecord = cjson.decode(record)
local want = decodedRecord["want"]
local batch = decodedRecord["batch"]

if redis.call("HEXISTS", keyWaits, "closed:" .. waitID) == 1 then
	redis.call("SREM", keyStepsPending, waitID)

	return { 2, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
end

redis.call("HSET", keyWaits, waitID, record)

-- A batch read is answered with the messages it wants saved since it last saw
-- the stack. It is answered at once if it never waits, if the SDK already holds
-- unread messages, or if there are any. Otherwise it stays recorded and a
-- message (or its timer) answers it.
if batch ~= nil then
	local ids = inbox_batch_ids(keyStep, keyStack, inbox, want, seen, batch["room"])

	if batch["drain"] or batch["unread"] or #ids > 0 then
		inbox_batch_save(keyStep, keyMetadata, keyStack, waitID, seen, ids)
		redis.call("SREM", keyStepsPending, waitID)

		return { 3, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
	end

	return { 0 }
end

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
		if ok and type(decoded) == "table" and decoded["inngest/inbox"] == inbox and inbox_wants(want, decoded) then
			redis.call("SREM", keyStepsPending, waitID)

			return { 1, stepID, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
		end
	end
end

return { 0 }
