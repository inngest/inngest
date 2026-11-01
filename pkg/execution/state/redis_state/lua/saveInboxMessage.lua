--[[

Saves a message step of an inbox, then finds the inbox's recorded wait. The wait
leaves the pending set; it stays recorded until the caller has queued its wake.

Saving the same step with the same value again saves nothing but still finds the
wait, so a redelivered event repairs a wake that was lost.

A message for a run that has ended, or for an inbox that was closed, is dropped.
A new message that would take the run past its step bound or its state size
bound is not saved. A bound of 0 means the bound is not checked.

A wait is satisfied only by a message after the step it last saw. A satisfied
batch read is also answered: a cut at the last message it covers is
saved under its id, unless it already has a result.

Output:
  { -1 }                         different data is already saved under the step
  { -3 }                         the run is at its step bound; nothing was saved
  { -4 }                         the message would pass the state size bound;
                                 nothing was saved
  { -5 }                         the run is gone; nothing was saved
  { -6 }                         the inbox is closed; nothing was saved
  { 0|-2 }                       saved (0) or already saved (-2); no wait
  { 0|-2, record, pending }      the same, and the satisfied wait's record and
                                 whether pending steps remain (1) or not (0)

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

-- $include(inbox_batch.lua)

local stepID = ARGV[1]
local value = ARGV[2]
local inbox = ARGV[3]
local maxSteps = tonumber(ARGV[4])
local maxState = tonumber(ARGV[5])

if redis.call("EXISTS", keyMetadata) == 0 then
	return { -5 }
end

-- The record that an inbox is closed. See closeInbox.lua.
if redis.call("HEXISTS", keyWaits, inbox .. ":closed") == 1 then
	return { -6 }
end

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

local record = redis.call("HGET", keyWaits, inbox)
if not record then
	return { status }
end

local wait = cjson.decode(record)

-- The wait is answered only if a message of the inbox is after the step it last
-- saw. A message saved again after the run moved on is not that message: the
-- wait recorded now is for a newer one.
--
-- A batch read is answered with a cut at the messages saved since it last saw
-- the stack, this one included.
if wait.batch ~= nil then
	local cut = inbox_cut(keyStep, keyStack, inbox, wait.seen, wait.batch.max)

	if cut == nil then
		return { status }
	end

	inbox_save_cut(keyStep, keyMetadata, keyStack, wait.id, cut)
elseif inbox_cut(keyStep, keyStack, inbox, wait.seen, 1) == nil then
	return { status }
end

redis.call("SREM", keyStepsPending, wait.id)

return { status, record, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
