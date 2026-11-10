-- Cuts through an inbox's messages: the last message step an answer covers.

-- The message step of an inbox that a cut falls on, among the steps saved after
-- the step `seen`. With a `max`, the cut is the max-th message after `seen`,
-- oldest first; without one it is the last message of the inbox. nil if there
-- is none. A step is a message of the inbox only if the executor wrote the
-- inbox's key in its value.
local function inbox_cut(keyStep, keyStack, inbox, seen, max)
	local function message(stepID)
		local value = redis.call("HGET", keyStep, stepID)

		if not value then
			return false
		end

		local ok, decoded = pcall(cjson.decode, value)

		return ok and type(decoded) == "table" and decoded["inngest/inbox"] == inbox
	end

	-- Walk back from the tail to the step the SDK last saw; those are the steps
	-- saved since, and the only ones worth reading.
	if max == nil then
		for i = redis.call("LLEN", keyStack) - 1, 0, -1 do
			local stepID = redis.call("LINDEX", keyStack, i)

			if stepID == seen then
				return nil
			end

			if message(stepID) then
				return stepID
			end
		end

		return nil
	end

	local gap = {}
	for i = redis.call("LLEN", keyStack) - 1, 0, -1 do
		local stepID = redis.call("LINDEX", keyStack, i)

		if stepID == seen then
			break
		end

		table.insert(gap, 1, stepID)
	end

	local found = nil
	local taken = 0
	for _, stepID in ipairs(gap) do
		if taken >= max then
			break
		end

		if message(stepID) then
			found = stepID
			taken = taken + 1
		end
	end

	return found
end

-- The value a cut is saved as.
local function inbox_cut_value(cut)
	if cut == nil then
		return '{"cut":null}'
	end

	return '{"cut":' .. cjson.encode(cut) .. "}"
end

-- Saves a cut under a step: the answer of a batch read or the result of a
-- close. The first answer wins, so a timeout or an earlier answer is kept.
local function inbox_save_cut(keyStep, keyMetadata, keyStack, stepID, cut)
	if redis.call("HEXISTS", keyStep, stepID) == 1 then
		return
	end

	local value = inbox_cut_value(cut)

	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, stepID, value)
	redis.call("RPUSH", keyStack, stepID)
end
