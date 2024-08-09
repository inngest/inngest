--[[]

Output:
    -1: Guaranteed capacity not found
    -2: Lease already exists
    -3: Invalid lease index
    0: Success

--]]

local keyGuaranteedCapacityMap   = KEYS[1]

local currentTimeMS           = tonumber(ARGV[1])
local guaranteedCapacityName  = ARGV[2]
local leaseID                 = ARGV[3]
local leaseIndex              = tonumber(ARGV[4])

-- $include(get_guaranteed_capacity_item.lua)
-- $include(decode_ulid_time.lua)

local guaranteedCapacity         = get_guaranteed_capacity_item(keyGuaranteedCapacityMap, guaranteedCapacityName)
if guaranteedCapacity == nil then
    return -1
end

-- TODO:
-- Filter expired leases based off of currentTimeMS
-- If index != remaining, fail.
-- Append lease to guaranteed capacity
-- Update map
-- Return OK

local validLeases = {}

if guaranteedCapacity.leases ~= nil and #guaranteedCapacity.leases > 0 then
    for _, lease in ipairs(guaranteedCapacity.leases) do
        if decode_ulid_time(lease) >= currentTimeMS then
            table.insert(validLeases, lease)
        end
    end
end

if leaseIndex < #validLeases then
    -- item is already leased due to contention:  someone asked for lease N, but
    -- lease N already exists.
    return -2
end

if leaseIndex ~= #validLeases then
    return -3
end

-- Add the new lease ID to valid leases, in effect garbage collecting expired leases.
table.insert(validLeases, leaseID)
guaranteedCapacity.leases = validLeases

redis.call("HSET", keyGuaranteedCapacityMap, guaranteedCapacityName, cjson.encode(guaranteedCapacity))

return 0
