--
-- activeCheckScanStatic loads a chunk of items from
-- the account active set (SET) and compares it to 2 static keys.
--
-- This script returns missing, leased, and stale (not found in either set) items.
-- It also returns a next cursor to continue scanning.
--

local keyActiveSet        = KEYS[1]
local keyStaticTarget1    = KEYS[2]
local keyStaticTarget2    = KEYS[3]
local keyQueueItemHash    = KEYS[4]

local cursor = tonumber(ARGV[1])
local batchSize = tonumber(ARGV[2])
local nowMS = tonumber(ARGV[3])

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
    if parsedData.leaseID ~= nil and parsedData.leaseID ~= false and parsedData.leaseID > nowMS then
      table.insert(leasedItems, itemID)
    else
      -- item may be stale: check all targets
      local targetScore1 = tonumber(redis.call("ZSCORE", keyStaticTarget1, itemID))
      if targetScore1 == nil or targetScore1 == false then
        local targetScore2 = tonumber(redis.call("ZSCORE", keyStaticTarget2, itemID))
        if targetScore2 == nil or targetScore2 == false then
          table.insert(staleItems, itemID)
        end
      end
    end
  end
end

return { nextCursor, missingItems, leasedItems, staleItems }
