-- Whether a read's filter wants a message. The filter is decoded JSON with two
-- optional parts: `src` (the message must come through one of these pauses) and
-- `events` (the message must be one of these event names). A message has to pass
-- every part that is there. No filter wants every message of the inbox.
local function inbox_wants(want, message)
	if type(want) ~= "table" then
		return true
	end

	if type(want["src"]) == "table" then
		local found = false

		for _, pause in ipairs(message["src"] or {}) do
			for _, wanted in ipairs(want["src"]) do
				if pause == wanted then
					found = true
				end
			end
		end

		if not found then
			return false
		end
	end

	if type(want["events"]) == "table" then
		local name = message["data"] and message["data"]["name"]
		local found = false

		for _, wanted in ipairs(want["events"]) do
			if name == wanted then
				found = true
			end
		end

		if not found then
			return false
		end
	end

	return true
end
