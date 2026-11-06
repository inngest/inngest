local function decodeManualOwner(value)
	local weight, request = string.match(value, "^(%d+):([^:]*)$")
	return tonumber(weight), request
end