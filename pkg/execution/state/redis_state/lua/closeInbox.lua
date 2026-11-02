--[[

Closes an inbox. Saves the close op's result, a cut at the last message the
inbox has, takes the close op out of the pending set, records that the inbox is
closed, then ends the inbox's wait. A wait that is recorded leaves the pending
set and is returned with its record, which stays until the caller has queued its
wake. The record of the close means a read that is recorded after this call
(the two ops can arrive in one response, in either order) is ended as it is
recorded, and a message that arrives after it is dropped.

Closing again saves nothing, and returns the wait if it is still recorded, so a
close that crashed before its wake was queued is repaired by running again.

Output:
  { pending }          pending is 1 when pending steps remain, else 0; no wait
  { pending, record }  the same, and the record of the wait that was ended

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

-- $include(inbox_batch.lua)

local stepID = ARGV[1]
local inbox = ARGV[2]

if redis.call("HEXISTS", keyStep, stepID) == 0 then
	inbox_save_cut(keyStep, keyMetadata, keyStack, stepID, inbox_cut(keyStep, keyStack, inbox, "", nil))
end

redis.call("SREM", keyStepsPending, stepID)

-- The record that the inbox is closed: the close op that closed it.
redis.call("HSET", keyWaits, inbox .. ":closed", stepID)

local record = redis.call("HGET", keyWaits, inbox)
if record then
	redis.call("SREM", keyStepsPending, cjson.decode(record).id)

	return { redis.call("SCARD", keyStepsPending) > 0 and 1 or 0, record }
end

return { redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
