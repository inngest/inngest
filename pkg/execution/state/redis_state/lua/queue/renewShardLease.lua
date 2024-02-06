--[[]

Output:
    -1: Shard not found
    -2: Lease not found
    0: success

--]]

local keyShardMap     = KEYS[1]

local currentTimeMS   = tonumber(ARGV[1])
local shardName       = ARGV[2]
local existingLeaseID = ARGV[3]
local newLeaseID      = ARGV[4]

-- $include(get_shard_item.lua)
-- $include(decode_ulid_time.lua)

local shard           = get_shard_item(keyShardMap, shardName)
if shard == nil then
    return -1
end

-- TODO:
-- Filter expired leases based off of currentTimeMS
-- If index != remaining, fail.
-- Append lease to shard
-- Update map
-- Return OK

local currentLeases = {}
local found = false

if shard.leases ~= nil and #shard.leases > 0 then
    for _, lease in ipairs(shard.leases) do
        if decode_ulid_time(lease) >= currentTimeMS then
            -- This is a valid lease.  If the lease matches what we're replacing,
            -- update the lease.
            if lease == existingLeaseID then
                table.insert(currentLeases, newLeaseID)
                found = true
            else
                table.insert(currentLeases, lease)
            end
        end
    end
end

if found == false then
    return -2
end

shard.leases = currentLeases
redis.call("HSET", keyShardMap, shardName, cjson.encode(shard))
return 0
