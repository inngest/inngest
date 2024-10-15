--[[]

Output:
    -1: Guaranteed capacity not found
    -2: Lease not found
    0: success
--]]

local keyGuaranteedCapacityMap  = KEYS[1]

local expire 										= tonumber(ARGV[1])
local currentTimeMS             = tonumber(ARGV[2])
local guaranteedCapacityName    = ARGV[3]
local existingLeaseID           = ARGV[4]
local newLeaseID                = ARGV[5]

-- $include(get_guaranteed_capacity_item.lua)
-- $include(decode_ulid_time.lua)

local guaranteedCapacity           = get_guaranteed_capacity_item(keyGuaranteedCapacityMap, guaranteedCapacityName)
if guaranteedCapacity == nil then
    return -1
end

local currentLeases = {}
local found = false

if guaranteedCapacity.leases ~= nil and #guaranteedCapacity.leases > 0 then
    local validLeases = {}
    for _, lease in ipairs(guaranteedCapacity.leases) do
        if decode_ulid_time(lease) >= currentTimeMS then
            -- This is a valid lease.  If the lease matches what we're replacing,
            -- update the lease.
            if lease == existingLeaseID then
                found = true
                if expire == 0 then
                  table.insert(validLeases, newLeaseID)
                end
			      else
                table.insert(validLeases, lease)
            end
        end
    end

    -- Filter out leases exceeding capacity
    for i, lease in ipairs(validLeases) do
        local lease_within_bounds = guaranteedCapacity.gc == nil or i <= tonumber(guaranteedCapacity.gc)
        if lease_within_bounds then
            table.insert(currentLeases, lease)
        end
    end
end

if found == false then
    return -2
end

guaranteedCapacity.leases = currentLeases
redis.call("HSET", keyGuaranteedCapacityMap, guaranteedCapacityName, cjson.encode(guaranteedCapacity))
return 0
