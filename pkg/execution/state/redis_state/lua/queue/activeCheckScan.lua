--
-- activeCheckScan loads a chunk of items from
-- the given active set (SET) and compares it to both
--
-- - the in progress set (ZSET)
-- - the respective partition ready queue (ZSET)
--
-- This script returns missing, leased, and stale (not found in either set) items.
-- It also returns a next cursor to continue scanning.
--

local keyActiveSet         = KEYS[1]
local keyInProgressZset    = KEYS[2]
local keyQueueItemHash     = KEYS[3]

local cursor = tonumber(ARGV[1])
local batchSize = tonumber(ARGV[2])
local nowMS = tonumber(ARGV[3])
local keyPrefix = ARGV[4]

-- $include(decode_ulid_time.lua)

local result = redis.call("SSCAN", keyActiveSet, cursor, "COUNT", batchSize)

local nextCursor = result[1]
local setMembers = result[2]

if #setMembers == 0 then
  return { nextCursor, {}, {}, {} }
end

local items = redis.call("HMGET", keyQueueItemHash, unpack(setMembers))

local missingItems  = {}
local leasedItems   = {}
local staleItems    = {}

for i = 1, #setMembers do
  local itemID = setMembers[i]
  local itemData = items[i]

  -- handle missing queue items
  if itemData == false or itemData == nil or itemData == "" then
    table.insert(missingItems, itemID)
  else
    local parsedData = cjson.decode(itemData)

    -- if item is still leased, ignore
    if parsedData.leaseID ~= nil and parsedData.leaseID ~= false and decode_ulid_time(parsedData.leaseID) > nowMS then
      table.insert(leasedItems, itemID)
    else
      -- item may be stale: check all targets
      local inProgressScore = tonumber(redis.call("ZSCORE", keyInProgressZset, itemID))
      if inProgressScore == nil or inProgressScore == false then
        -- retrieve partition ID
        local partitionID = parsedData.wfID
        if parsedData.queueID ~= false and parsedData.queueID ~= nil then
          partitionID = parsedData.queueID
        end

        local keyReadyPartition = string.format("%s:queue:sorted:%s", keyPrefix, partitionID)

        local partitionScore = tonumber(redis.call("ZSCORE", keyReadyPartition, itemID))
        if partitionScore == nil or partitionScore == false then
          table.insert(staleItems, itemData)
        end
      end
    end
  end
end

return { nextCursor, missingItems, leasedItems, staleItems }
