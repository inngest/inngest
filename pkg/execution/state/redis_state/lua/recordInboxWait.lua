--[[

Records the wait of an inbox, then scans the steps saved after the step the wait
last saw. If one of them is a message of the inbox, the wait is satisfied: it
leaves the pending set but stays recorded until the caller has queued its wake.

An inbox has one reader, so it has at most one wait, recorded under the inbox's
id.

The record is written before the scan, and an arrival saves before it looks at
the record, so at least one of the two always sees the other.

A read of an inbox that was closed is not recorded: it leaves the pending set
and is ended.

A read that the response put in the pending set (it carries a coalesce key) and
that is no longer in it was ended by another path, which took it out and owns
the run's continuation. The SDK reports a pending read again in the response of
a sibling step, so this can be that report arriving late. It is not recorded and
wakes nothing: a second wake would run every step after the read twice.

Output:
  { 0 }                           recorded; nothing for it to read
  { 1, messageStepID, pending }   recorded and satisfied; pending is 1 when
                                  pending steps remain, else 0
  { 2, pending }                  not recorded; its inbox was closed
  { 3, pending }                  a batch read, recorded and answered with a cut
  { 4 }                           not recorded; another path already ended it

]]

local keyStep = KEYS[1]
local keyStack = KEYS[2]
local keyStepsPending = KEYS[3]
local keyWaits = KEYS[4]
local keyMetadata = KEYS[5]

-- $include(inbox_batch.lua)

local waitID = ARGV[1]
local record = ARGV[2]
local inbox = ARGV[3]
local seen = ARGV[4]

local batch = cjson.decode(record)["batch"]

-- The record that an inbox is closed. See closeInbox.lua.
if redis.call("HEXISTS", keyWaits, inbox .. ":closed") == 1 then
	redis.call("SREM", keyStepsPending, waitID)

	return { 2, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
end

if cjson.decode(record)["pck"] ~= nil and redis.call("SISMEMBER", keyStepsPending, waitID) == 0 then
	-- A batch read's answer is saved under its own id. If it is there, this
	-- call is the same one sent again after its response was lost, and nobody
	-- has queued the wake yet: answer it as satisfied.
	if batch ~= nil and redis.call("HEXISTS", keyStep, waitID) == 1 then
		return { 3, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
	end

	return { 4 }
end

redis.call("HSET", keyWaits, inbox, record)

-- A batch read is answered with a cut at the messages saved since it last saw
-- the stack. It is answered at once if it never waits, if the SDK already holds
-- unread messages, or if there are any. Otherwise it stays recorded and a
-- message (or its timer) answers it.
if batch ~= nil then
	local cut = inbox_cut(keyStep, keyStack, inbox, seen, batch["max"])

	-- Unread messages the SDK holds are before the step it saw.
	if cut == nil and batch["unread"] then
		cut = inbox_cut(keyStep, keyStack, inbox, "", nil)
	end

	if batch["drain"] or batch["unread"] or cut ~= nil then
		inbox_save_cut(keyStep, keyMetadata, keyStack, waitID, cut)
		redis.call("SREM", keyStepsPending, waitID)

		return { 3, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
	end

	return { 0 }
end

local first = inbox_cut(keyStep, keyStack, inbox, seen, 1)

if first ~= nil then
	redis.call("SREM", keyStepsPending, waitID)

	return { 1, first, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
end

return { 0 }
