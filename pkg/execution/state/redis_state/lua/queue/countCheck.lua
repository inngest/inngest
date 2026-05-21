--[[

  Inspect and retrieve counters related to a partition

]]
local keyAccountInProgress = KEYS[1]
local keyReady             = KEYS[2]
local keyInProgress        = KEYS[3]
local keyShadowPartition   = KEYS[4]

local nowMS = ARGV[1]

local acct_in_progress = redis.call("ZCARD", keyAccountInProgress)

local ready = redis.call("ZCARD", keyReady)
local in_progress = redis.call("ZCOUNT", keyInProgress, nowMS, "+inf")
local future = redis.call("ZCOUNT", keyReady, nowMS, "+inf")

local backlogs = redis.call("ZCARD", keyShadowPartition)

return cjson.encode({
    acct_in_progress = acct_in_progress,
    ready = ready,
    in_progress = in_progress,
    future = future,
    backlogs = backlogs
})
