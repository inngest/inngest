--[[

  Normalize the items in the backlog by requeueing them into
  the appropriate one.

  return a result of the normalization process
  - number requeued
  - total
  - remaining
]]

local keyBacklogSet = KEYS[1]

local backlogID = ARGV[1]
local limit = tonumber(ARGV[2])


-- Get the number of items remaining in the backlog
local count = redis.call("ZCARD", keyBacklogSet)

-- retrieve the max number of items from the backlog
local items = redis.call("ZRANGE", "-inf", "+inf", "BYSCORE", limit)

-- TODO
-- enqueue it to the appropriate backlog
-- return the result
