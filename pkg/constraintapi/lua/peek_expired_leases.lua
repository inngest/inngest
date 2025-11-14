---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, unpack(arg))
end

---@type string[]
local KEYS = KEYS

---@type string[]
local ARGV = ARGV

local keyAccountLeases = KEYS[1]

local nowMS = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])

local offset = 0

local count = call("ZCOUNT", keyAccountLeases, "-inf", tostring(nowMS))
if count > limit then
	math.randomseed(tonumber(nowMS))
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count - limit) + 1) - 1
end

local leaseIDs = call("ZRANGE", keyAccountLeases, "-inf", tostring(nowMS), "BYSCORE", "LIMIT", offset, limit)
if #leaseIDs == 0 then
	return {}
end

return { count, leaseIDs }
