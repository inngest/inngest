-- One record contains both the run's held weight and its active request ID.
-- An empty request ID permits a retry; zero weight is a finalized tombstone.
local function decodeManualOwner(value)
	local weight, request = string.match(value, "^(%d+):([^:]*)$")
	return tonumber(weight), request
end
