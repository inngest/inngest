--[[

Removes account pointer from global accounts ZSET if account partitions ZSET is empty

NOTE: This is only required while we're running old, pre-key-queue/account-queue code, and can be removed
once the system is fully rolled out. This is because old code doesn't properly clean up unknown keys
including account partitions, global accounts, and concurrency key queues.

]]

local keyGlobalAccountPointer = KEYS[1]
local keyAccountPartitions    = KEYS[2]

local accountId = ARGV[1]

if tonumber(redis.call("ZCARD", keyAccountPartitions)) > 0 then
	return -1 -- not actually empty
end

-- no account partitions: drop account pointer from global accounts ZSET
redis.call("ZREM", keyGlobalAccountPointer, accountId)

return 1
