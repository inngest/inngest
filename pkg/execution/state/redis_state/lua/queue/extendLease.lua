--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local keyQueueMap       = KEYS[1] -- queue:item - hash: { $itemID: item }

local keyPartitionA     = KEYS[2]           -- queue:sorted:$workflowID - zset
local keyPartitionB     = KEYS[3]           -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC     = KEYS[4]          -- e.g. sorted:c|t:$workflowID - zset

local keyConcurrencyA    = KEYS[5] -- Account concurrency level
local keyConcurrencyB    = KEYS[6] -- When leasing an item we need to place the lease into this key.
local keyConcurrencyC    = KEYS[7] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency = KEYS[8]       

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]

-- $include(decode_ulid_time.lua)
-- $include(get_queue_item.lua)

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)

local function ends_with(str, ending)
   return ending == "" or str:sub(-#ending) == ending
end

-- Look up the current queue item.  We need to see if the queue item is already leased.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
	return 1
end
if item.leaseID == nil or item.leaseID == cjson.null then
	return 2
end
if item.leaseID ~= currentLeaseKey then
	return 3
end

item.leaseID = newLeaseKey
-- Update the item's lease key.
redis.call("HSET", keyQueueMap, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.

-- Items always belong to an account
redis.call("ZADD", keyAcctConcurrency, nextTime, item.id)

if keyConcurrencyA ~= nil and keyConcurrencyA ~= "" and ends_with(keyConcurrencyA, "sorted:-") ~= true then
	redis.call("ZADD", keyConcurrencyA, nextTime, item.id)
end
if keyConcurrencyB ~= nil and keyConcurrencyB ~= "" and ends_with(keyConcurrencyB, "sorted:-") ~= true then
	redis.call("ZADD", keyConcurrencyB, nextTime, item.id)
end
if keyConcurrencyC ~= nil and keyConcurrencyC ~= "" and ends_with(keyConcurrencyC, "sorted:-") ~= true then
	redis.call("ZADD", keyConcurrencyC, nextTime, item.id)
end

return 0
