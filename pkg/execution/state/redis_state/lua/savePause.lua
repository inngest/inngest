-- [[
--
-- Output:
--   [1..N]: Successfully saved pause;  returns # of pauses in AddIdx
--   -1: Pause already exists
-- ]]

local pauseKey    = KEYS[1]
local pauseEvtKey = KEYS[2]
local pauseInvokeKey = KEYS[3]
local pauseSignalKey = KEYS[4]
local keyPauseAddIdx = KEYS[5]
local keyPauseExpIdx = KEYS[6]
local keyRunPauses   = KEYS[7]
local keyPausesIdx   = KEYS[8]

local pause          = ARGV[1]
local pauseID        = ARGV[2]
local event          = ARGV[3]
local invokeCorrelationID = ARGV[4]
local signalCorrelationID = ARGV[5]
local extendedExpiry = tonumber(ARGV[6])
local nowUnixSeconds = tonumber(ARGV[7])
local canReplaceSignal = tonumber(ARGV[8])


if redis.call("SETNX", pauseKey, pause) == 0 then
	return -1
end

-- Populate global index
redis.call("SADD", keyPausesIdx, pauseID)

redis.call("EXPIRE", pauseKey, extendedExpiry)

-- Add an index of when the pause was added.
redis.call("ZADD", keyPauseAddIdx, nowUnixSeconds, pauseID)
-- Add an index of when the pause expires.  This lets us manually
-- garbage collect expired pauses from the HSET below.
redis.call("ZADD", keyPauseExpIdx, nowUnixSeconds+extendedExpiry, pauseID)

-- SADD to store the pause for this run
redis.call("SADD", keyRunPauses, pauseID)

if event ~= false and event ~= "" and event ~= nil then
	redis.call("HSET", pauseEvtKey, pauseID, pause)
end

if invokeCorrelationID ~= false and invokeCorrelationID ~= "" and invokeCorrelationID ~= nil then
	redis.call("HSETNX", pauseInvokeKey, invokeCorrelationID, pauseID)
end

if signalCorrelationID ~= false and signalCorrelationID ~= "" and signalCorrelationID ~= nil then
	if canReplaceSignal == 1 then
		-- Note that this may be overwriting an existing signal wait, which is
		-- intentional. When running this transaction, we could be part of an
		-- idempotent retry or be overwriting an existing signal wait with
		-- this one. In the latter case, it is expected that the previous
		-- signal wait will be left to reach its timeout. We also leave the
		-- pause behind, as it may also be in a different block.
		redis.call("HSET", pauseSignalKey, signalCorrelationID, pauseID)
	else
		if redis.call("HSETNX", pauseSignalKey, signalCorrelationID, pauseID) == 0 then
			-- The signal already exists! The rarer case now is that this is
			-- an idempotent retry for saving a pause, so let's check if
			-- we're trying to save this pause for the same run / step ID. If
			-- not, we need to return an error to roll back all of these
			-- changes, as we're trying to duplicate a signal.
			local existing = redis.call("HGET", pauseSignalKey, signalCorrelationID)
			if existing ~= pauseID then
				return redis.error_reply("ErrSignalConflict")
			end
		end
	end
end


return redis.call("ZCARD", keyPauseAddIdx)
