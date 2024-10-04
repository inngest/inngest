--[[]

Output:
    -1: Guaranteed capacity not found
    -2: Lease not found
    0: success
--]]

local keyGuaranteedCapacityMap  = KEYS[1]

local currentTimeMS             = tonumber(ARGV[1])
local guaranteedCapacityName    = ARGV[2]
local existingLeaseID           = ARGV[3]
local newLeaseID                = ARGV[4]

-- $include(get_guaranteed_capacity_item.lua)
-- $include(decode_ulid_time.lua)

local guaranteedCapacity           = get_guaranteed_capacity_item(keyGuaranteedCapacityMap, guaranteedCapacityName)
if guaranteedCapacity == nil then
    return -1
end

-- TODO:
-- Filter expired leases based off of currentTimeMS
-- If index != remaining, fail.
-- Append lease to guaranteed capacity
-- Update map
-- Return OK

local currentLeases = {}
local found = false

if guaranteedCapacity.leases ~= nil and #guaranteedCapacity.leases > 0 then
    for _, lease in ipairs(guaranteedCapacity.leases) do
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

guaranteedCapacity.leases = currentLeases
redis.call("HSET", keyGuaranteedCapacityMap, guaranteedCapacityName, cjson.encode(guaranteedCapacity))
return 0
