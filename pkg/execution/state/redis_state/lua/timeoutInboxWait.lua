--[[

Times a recorded inbox wait out. If the wait is still recorded, saves the timeout
value under the wait's id, takes the wait out of the pending set and returns its
record. The record stays until the caller has queued the wake. If the wait is
not recorded, a message or a close already took it and nothing changes.

Saving again, with the wait still recorded, saves nothing and returns the record
again, so a timer that runs twice repairs a lost wake.

Output:
  { 0 }                    the wait is not recorded
  { 1, record, pending }   timed out; pending is 1 when pending steps remain,
                           else 0

]]

local keyStep = KEYS[1]
local keyMetadata = KEYS[2]
local keyStack = KEYS[3]
local keyStepsPending = KEYS[4]
local keyWaits = KEYS[5]

local waitID = ARGV[1]
local value = ARGV[2]

local record = redis.call("HGET", keyWaits, waitID)
if not record then
	return { 0 }
end

if redis.call("HEXISTS", keyStep, waitID) == 0 then
	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, waitID, value)
	redis.call("RPUSH", keyStack, waitID)
end

redis.call("SREM", keyStepsPending, waitID)

return { 1, record, redis.call("SCARD", keyStepsPending) > 0 and 1 or 0 }
