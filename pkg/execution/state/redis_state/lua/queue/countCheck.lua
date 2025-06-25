--[[

  Inspect and retrieve counters related to a partition

]]
local keyAccountActive     = KEYS[1]
local keyAccountInProgress = KEYS[2]
local keyReady             = KEYS[3]
local keyInProgress        = KEYS[4]
local keyActive            = KEYS[5]
local keyShadowPartition   = KEYS[6]

local nowMS = ARGV[1]

local acct_active = redis.call("SCARD", keyAccountActive)
local acct_in_progress = redis.call("ZCARD", keyAccountInProgress)

local ready = redis.call("ZCARD", keyReady)
local in_progress = redis.call("ZCOUNT", keyInProgress, nowMS, "+inf")
local active = redis.call("SCARD", keyActive)
local future = redis.call("ZCOUNT", keyReady, nowMS, "+inf")

local backlogs = redis.call("ZCARD", keyShadowPartition)

return cjson.encode({
    acct_active = acct_active,
    acct_in_progress = acct_in_progress,
    ready = ready,
    in_progress = in_progress,
    active = active,
    future = future,
    backlogs = backlogs
})
