-- The messages a batch read is answered with. Needs inbox_wants.

-- The step ids of the messages of an inbox that a filter wants, saved after the
-- step `seen`, oldest first. `room` is how many may be returned; nil means all.
local function inbox_batch_ids(keyStep, keyStack, inbox, want, seen, room)
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

	local ids = {}
	for _, stepID in ipairs(gap) do
		if room ~= nil and #ids >= room then
			break
		end

		local value = redis.call("HGET", keyStep, stepID)

		if value then
			local ok, decoded = pcall(cjson.decode, value)

			-- Only the executor writes this key, in the value of a message step.
			if ok and type(decoded) == "table" and decoded["inngest/inbox"] == inbox and inbox_wants(want, decoded) then
				table.insert(ids, stepID)
			end
		end
	end

	return ids
end

-- Saves the answer of a batch read under the read's op: the step it was
-- answered after and the message step ids. The first answer wins, so a timeout
-- or an earlier answer is kept.
local function inbox_batch_save(keyStep, keyMetadata, keyStack, waitID, seen, ids)
	if redis.call("HEXISTS", keyStep, waitID) == 1 then
		return
	end

	local quoted = {}
	for _, stepID in ipairs(ids) do
		table.insert(quoted, cjson.encode(stepID))
	end

	local value = '{"seen":' .. cjson.encode(seen) .. ',"ids":[' .. table.concat(quoted, ",") .. "]}"

	redis.call("HINCRBY", keyMetadata, "state_size", #value)
	redis.call("HINCRBY", keyMetadata, "step_count", 1)
	redis.call("HSET", keyStep, waitID, value)
	redis.call("RPUSH", keyStack, waitID)
end
