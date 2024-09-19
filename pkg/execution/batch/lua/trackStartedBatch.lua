--
--This script tracks started batches for each account
--

local keyPendingBatchCount = KEYS[1] -- key to the pending batch count

local accountId = ARGV[1] -- the account ID

redis.call("HINCRBY", keyPendingBatchCount, accountId, -1)

return 0
