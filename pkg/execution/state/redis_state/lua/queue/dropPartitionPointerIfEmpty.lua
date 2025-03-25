--[[
  Atomically drops partition pointer from index if partition is empty.
]]

local keyIndex = KEYS[1]
local keyPartition = KEYS[2]

local pointer = ARGV[1]

local count = tonumber(redis.call("ZCARD", keyPartition))
if count == 0 then
  redis.call("ZREM", keyIndex, pointer)
  return 1
end

return 0
