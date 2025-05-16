--[[

Requeues a job by its given ID.  This returns an error if the job
does not exist within the queue index (outstanding queue).

NOTE: This is used by debounce to push back the timeout job. It is not related to Requeue() which moves an in-progress item back to the backlog/queue.

Return values:

- 0:  Successfully requeued
- -1: Queue item not found
- -2: Queue item is leased and being worked on.

]]
--

local keyQueueHash            = KEYS[1] -- queue:item - hash
local keyPartitionMap         = KEYS[2] -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        = KEYS[3] -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[4] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[5] -- accounts:$accountId:partition:sorted

local keyPartitionFn    = KEYS[6] -- queue:sorted:$workflowID - zset

local jobID            = ARGV[1]           -- queue item ID
local jobScore         = tonumber(ARGV[2]) -- enqueue at, in milliseconds
local nowMS            = tonumber(ARGV[3]) -- in ms
local partitionItem    = ARGV[4]
local partitionID      = ARGV[5]
local accountID        = ARGV[6]

-- $include(decode_ulid_time.lua)
-- $include(get_queue_item.lua)
-- $include(update_pointer_score.lua)
-- $include(get_partition_item.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(enqueue_to_partition.lua)

local item = get_queue_item(keyQueueHash, jobID)
if item == nil then
    return -1
end

-- Ensure that we're not requeueing a leased job.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > nowMS then
    -- This is already leased, so don't requeue by ID.  Use the standard requeue operation.
    return -2
end


-- Update the "at" time of the job
item.at = jobScore
item.wt = jobScore
redis.call("HSET", keyQueueHash, jobID, cjson.encode(item))

requeue_to_partition(keyPartitionFn, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, jobScore, jobID, nowMS, accountID)

return 0
