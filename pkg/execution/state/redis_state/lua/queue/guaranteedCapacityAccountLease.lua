--[[]

Output:
    -1: Guaranteed capacity not found
    -2: Lease already exists (tried to overwrite a valid, existing lease)
    -3: Invalid lease index (tried to lease index other than the next one)
    -4: Exceeded guaranteed capacity (tried to lease same account more often than allowed)
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

local validLeases = {}

-- Filter out expired leases
if guaranteedCapacity.leases ~= nil and #guaranteedCapacity.leases > 0 then
    for _, lease in ipairs(guaranteedCapacity.leases) do
        if decode_ulid_time(lease) >= currentTimeMS then
            table.insert(validLeases, lease)
        end
    end
end

-- Prevent leasing already-leased index
if leaseIndex < #validLeases then
    -- item is already leased due to contention:  someone asked for lease N, but
    -- lease N already exists and is still valid.
    return -2
end

-- Prevent leasing higher index than guaranteed capacity (invariant must hold: index < guaranteedCapacity.gc)
-- This is a sanity check and not strictly required, but this case should never happen.
if guaranteedCapacity.gc ~= nil and leaseIndex >= tonumber(guaranteedCapacity.gc) then
		return -4
end

-- Prevent skipping index (must lease immediate next index)
if leaseIndex ~= #validLeases then
    return -3
end

-- Add the new lease ID to valid leases, in effect garbage collecting expired leases.
table.insert(validLeases, leaseID)
guaranteedCapacity.leases = validLeases

redis.call("HSET", keyGuaranteedCapacityMap, guaranteedCapacityName, cjson.encode(guaranteedCapacity))

return 0
